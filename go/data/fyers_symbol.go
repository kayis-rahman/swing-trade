package data

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// FyersSymbolMasterCSVURL is the unauthenticated public Fyers symbol master,
// matching Java's FyersSymbolMasterService.CSV_URL.
const FyersSymbolMasterCSVURL = "https://public.fyers.in/sym_details/NSE_CM.csv"

// fyersSymbolMasterDownloadLimit matches Java's16 MiB WebClient codec limit.
const fyersSymbolMasterDownloadLimit = 16 << 20

// Headerless CSV columns (verified against a live download2026-07-04):
// 0 fyToken | 1 name | 2 exch instrument type | 3 lot size | 4 tick size | 5 ISIN
// 6 trading session | 7 last update | 8 expiry | 9 fyers symbol (NSE:X-EQ)
// 10 exchange id | 11 segment id | 12 scrip code | 13 trading symbol (X)
const (
	fyersColumnFyToken       = 0
	fyersColumnName          = 1
	fyersColumnLotSize       = 3
	fyersColumnTickSize      = 4
	fyersColumnISIN          = 5
	fyersColumnFyersSymbol   = 9
	fyersColumnTradingSymbol = 13
	fyersMinimumColumns      = 14
)

var fyersEqSymbol = regexp.MustCompile(`^NSE:.+-EQ$`)

// FyersSymbol mirrors the fyers_symbol_master row consumed by symbol search
// and Fyers instrument lookups. The segment column is never populated by the
// CSV refresh and stays null, as in Java.
type FyersSymbol struct {
	FyToken       string
	FyersSymbol   string
	TradingSymbol string
	Name          *string
	Exchange      string
	LotSize       *int32
	TickSize      *string
	ISIN          *string
	UpdatedAt     time.Time
}

// ParseFyersSymbolMasterCSV maps one headerless download to equity rows the
// same way Java's FyersSymbolMasterService.parse does: short rows and
// unparseable rows count as skipped, non-EQ rows are silently dropped, and a
// mid-stream parse failure returns the rows read so far.
func ParseFyersSymbolMasterCSV(contents string, now time.Time) (rows []FyersSymbol, skipped int) {
	reader := csv.NewReader(strings.NewReader(contents))
	reader.FieldsPerRecord = -1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Java catches the parser failure, logs it, and keeps the rows parsed so far.
			return rows, skipped
		}
		if len(record) < fyersMinimumColumns {
			skipped++
			continue
		}
		fyersSymbol := strings.TrimSpace(record[fyersColumnFyersSymbol])
		if !fyersEqSymbol.MatchString(fyersSymbol) {
			continue
		}
		row := FyersSymbol{
			FyToken:       strings.TrimSpace(record[fyersColumnFyToken]),
			FyersSymbol:   fyersSymbol,
			TradingSymbol: strings.TrimSpace(record[fyersColumnTradingSymbol]),
			Name:          stringPtr(strings.TrimSpace(record[fyersColumnName])),
			Exchange:      "NSE",
			LotSize:       parseFyersLotSize(record[fyersColumnLotSize]),
			TickSize:      parseFyersTickSize(record[fyersColumnTickSize]),
			ISIN:          blankToNil(strings.TrimSpace(record[fyersColumnISIN])),
			UpdatedAt:     now,
		}
		if row.FyToken == "" || row.TradingSymbol == "" {
			skipped++
			continue
		}
		rows = append(rows, row)
	}
	return rows, skipped
}

// parseFyersLotSize mirrors Java's parseIntSafe: Double-style parsing truncated
// to an integer, with NaN, infinities, and out-of-range values treated as null
// instead of Java's zero/saturating int cast.
func parseFyersLotSize(value string) *int32 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed > float64(1<<31-1) || parsed < -float64(1<<31) {
		return nil
	}
	truncated := int32(parsed)
	return &truncated
}

// parseFyersTickSize mirrors Java's parseDecimalSafe with PostgreSQL's
// numeric text format as the validator.
func parseFyersTickSize(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	var numeric pgtype.Numeric
	if err := numeric.Scan(trimmed); err != nil {
		return nil
	}
	return &trimmed
}

func blankToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string { return &value }

// FyersSymbolWriteDB is the connection surface the wipe-and-replace refresh needs.
type FyersSymbolWriteDB interface {
	sqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// FyersSymbolStore reads and transactionally replaces the symbol master.
type FyersSymbolStore struct {
	db      FyersSymbolWriteDB
	queries *sqlc.Queries
}

// NewFyersSymbolStore creates a store on a pgx-compatible pool or transaction handle.
func NewFyersSymbolStore(database FyersSymbolWriteDB) *FyersSymbolStore {
	return &FyersSymbolStore{db: database, queries: sqlc.New(database)}
}

// ReplaceFyersSymbols deletes every stored symbol and inserts the parsed rows
// in one transaction, matching Java's @Transactional deleteAllInBatch + saveAll.
func (store *FyersSymbolStore) ReplaceFyersSymbols(ctx context.Context, rows []FyersSymbol) error {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin symbol master replacement: %w", err)
	}
	queries := sqlc.New(tx)
	if err := queries.DeleteAllFyersSymbols(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete stored symbols: %w", err)
	}
	for _, row := range rows {
		params := sqlc.InsertFyersSymbolParams{
			FyToken:       row.FyToken,
			FyersSymbol:   row.FyersSymbol,
			TradingSymbol: row.TradingSymbol,
			Name:          toText(row.Name),
			Exchange:      row.Exchange,
			LotSize:       toInt4(row.LotSize),
			Isin:          toText(row.ISIN),
			UpdatedAt:     pgtype.Timestamp{Time: row.UpdatedAt, Valid: true},
		}
		if row.TickSize != nil {
			if err := params.TickSize.Scan(*row.TickSize); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("encode tick size %q: %w", *row.TickSize, err)
			}
		}
		if err := queries.InsertFyersSymbol(ctx, params); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("insert symbol %s: %w", row.FyToken, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit symbol master replacement: %w", err)
	}
	return nil
}

// CountFyersSymbols returns the stored symbol-master row count.
func (store *FyersSymbolStore) CountFyersSymbols(ctx context.Context) (int64, error) {
	count, err := store.queries.CountFyersSymbols(ctx)
	if err != nil {
		return 0, fmt.Errorf("count Fyers symbols: %w", err)
	}
	return count, nil
}

// SearchFyersSymbols matches Java's repository search: prefix match on the
// trading symbol or case-insensitive contains match on the name, ordered by
// trading symbol and limited to the requested page size.
func (store *FyersSymbolStore) SearchFyersSymbols(ctx context.Context, query string, limit int32) ([]FyersSymbol, error) {
	rows, err := store.queries.SearchFyersSymbols(ctx, sqlc.SearchFyersSymbolsParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search Fyers symbols: %w", err)
	}
	return mapFyersSymbols(rows), nil
}

// FindFyersSymbolsByTradingSymbol mirrors findByTradingSymbolIgnoreCase; the
// caller decides how to treat the rare duplicate trading symbols.
func (store *FyersSymbolStore) FindFyersSymbolsByTradingSymbol(ctx context.Context, symbol string) ([]FyersSymbol, error) {
	rows, err := store.queries.FindFyersSymbolsByTradingSymbol(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("find Fyers symbol %s: %w", symbol, err)
	}
	return mapFyersSymbols(rows), nil
}

// ListFyersTradingSymbols returns every stored trading symbol, ordered for
// deterministic Go output where Java's findAll order was unspecified.
func (store *FyersSymbolStore) ListFyersTradingSymbols(ctx context.Context) ([]string, error) {
	symbols, err := store.queries.ListFyersTradingSymbols(ctx)
	if err != nil {
		return nil, fmt.Errorf("list Fyers trading symbols: %w", err)
	}
	return symbols, nil
}

// ListFyersSymbolsByExchange mirrors findByExchangeIgnoreCaseOrderByTradingSymbolAsc.
func (store *FyersSymbolStore) ListFyersSymbolsByExchange(ctx context.Context, exchange string) ([]FyersSymbol, error) {
	rows, err := store.queries.ListFyersSymbolsByExchange(ctx, exchange)
	if err != nil {
		return nil, fmt.Errorf("list Fyers symbols for exchange %s: %w", exchange, err)
	}
	return mapFyersSymbols(rows), nil
}

func mapFyersSymbols(rows []sqlc.FyersSymbolMaster) []FyersSymbol {
	symbols := make([]FyersSymbol, 0, len(rows))
	for _, row := range rows {
		symbols = append(symbols, FyersSymbol{
			FyToken:       row.FyToken,
			FyersSymbol:   row.FyersSymbol,
			TradingSymbol: row.TradingSymbol,
			Name:          fromText(row.Name),
			Exchange:      row.Exchange,
			LotSize:       fromInt4(row.LotSize),
			TickSize:      fromNumeric(row.TickSize),
			ISIN:          fromText(row.Isin),
			UpdatedAt:     row.UpdatedAt.Time,
		})
	}
	return symbols
}

func toText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func fromText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func toInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func fromInt4(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func fromNumeric(value pgtype.Numeric) *string {
	if !value.Valid {
		return nil
	}
	raw, err := value.Value()
	if err != nil {
		return nil
	}
	text, ok := raw.(string)
	if !ok {
		return nil
	}
	return &text
}

// FyersSymbolMasterClient downloads the unauthenticated public symbol master
// with Java's16 MiB in-memory ceiling.
type FyersSymbolMasterClient struct {
	httpClient *http.Client
	url        string
}

// NewFyersSymbolMasterClient constructs a downloader; a nil client gets a
// 15-second timeout like the other Go market-data clients.
func NewFyersSymbolMasterClient(httpClient *http.Client) *FyersSymbolMasterClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &FyersSymbolMasterClient{httpClient: httpClient, url: FyersSymbolMasterCSVURL}
}

// Download returns the raw CSV body or an error; oversized bodies are refused
// before they are buffered.
func (client *FyersSymbolMasterClient) Download(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.url, nil)
	if err != nil {
		return "", fmt.Errorf("build symbol master request: %w", err)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("download Fyers symbol master: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return "", fmt.Errorf("download Fyers symbol master: unexpected status %d", response.StatusCode)
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, fyersSymbolMasterDownloadLimit+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return "", fmt.Errorf("read fyers symbol master: %w", readErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close fyers symbol master response: %w", closeErr)
	}
	if int64(len(body)) > fyersSymbolMasterDownloadLimit {
		return "", fmt.Errorf("fyers symbol master exceeds the %d byte limit", fyersSymbolMasterDownloadLimit)
	}
	return string(body), nil
}

// FyersSymbolCSVDownloader supplies the raw symbol master CSV.
type FyersSymbolCSVDownloader interface {
	Download(ctx context.Context) (string, error)
}

// FyersSymbolMaster downloads, parses, and transactionally replaces the
// symbol master, mirroring Java's FyersSymbolMasterService.refresh.
type FyersSymbolMaster struct {
	client FyersSymbolCSVDownloader
	store  FyersSymbolReplacer
	now    func() time.Time
	logger *slog.Logger
}

// FyersSymbolReplacer is the persistence surface the refresh requires.
type FyersSymbolReplacer interface {
	ReplaceFyersSymbols(ctx context.Context, rows []FyersSymbol) error
	CountFyersSymbols(ctx context.Context) (int64, error)
}

// NewFyersSymbolMaster wires the refresh over explicit ports.
func NewFyersSymbolMaster(client FyersSymbolCSVDownloader, store FyersSymbolReplacer, now func() time.Time, logger *slog.Logger) (*FyersSymbolMaster, error) {
	if client == nil || store == nil || now == nil || logger == nil {
		return nil, fmt.Errorf("client, store, clock, and logger are required")
	}
	return &FyersSymbolMaster{client: client, store: store, now: now, logger: logger}, nil
}

// Refresh downloads the master and replaces the stored rows. A failed or empty
// download never reaches the wipe, matching Java's early return, while a
// persistence failure aborts the transaction.
func (master *FyersSymbolMaster) Refresh(ctx context.Context) (int, error) {
	contents, err := master.client.Download(ctx)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(contents) == "" {
		return 0, fmt.Errorf("fyers symbol master download returned empty response")
	}
	rows, skipped := ParseFyersSymbolMasterCSV(contents, master.now())
	if skipped > 0 {
		master.logger.Warn("skipped malformed rows while parsing Fyers symbol master", "module", "data", "skipped", skipped)
	}
	if err := master.store.ReplaceFyersSymbols(ctx, rows); err != nil {
		return 0, err
	}
	master.logger.Info("refreshed Fyers symbol master", "module", "data", "symbols", len(rows))
	return len(rows), nil
}
