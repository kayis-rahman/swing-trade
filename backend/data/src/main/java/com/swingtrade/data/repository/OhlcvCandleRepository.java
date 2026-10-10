package com.swingtrade.data.repository;

import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.service.Interval;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.List;
import java.util.Optional;

/**
 * Repository interface for OhlcvCandleEntity operations.
 * Optimized for time-series queries on TimescaleDB hypertable.
 *
 * <p><b>Interval contract:</b> every pre-existing query is daily-mode — it filters
 * {@code interval = 'D'} implicitly so it returns exactly the rows it returned before
 * the interval dimension existed, even when intraday bars are stored for the same
 * symbol and date. Intraday access goes through the explicit {@code …AndInterval}
 * methods. The storage code for daily rows is {@link Interval#DAILY}'s code ("D").
 */
@Repository
public interface OhlcvCandleRepository extends JpaRepository<OhlcvCandleEntity, Long> {

    /**
     * Storage code of the daily interval; every daily-mode query filters on it.
     * A compile-time constant so it can be inlined into {@code @Query} annotations;
     * {@link Interval#DAILY} must always return this code (asserted by tests).
     */
    String DAILY_INTERVAL = "D";

    @Modifying
    @Query(value = "INSERT INTO ohlcv_candles (symbol, timeframe, date, bar_time, open_price, high_price, low_price, close_price, volume, adj_close_price) "
        + "VALUES (:symbol, :interval, :date, :barTime, :open, :high, :low, :close, :volume, :adjClose) "
        + "ON CONFLICT (symbol, timeframe, date, bar_time) DO NOTHING", nativeQuery = true)
    int insertIfAbsent(@Param("symbol") String symbol,
                       @Param("interval") String interval,
                       @Param("date") LocalDate date,
                       @Param("barTime") LocalTime barTime,
                       @Param("open") java.math.BigDecimal open, @Param("high") java.math.BigDecimal high,
                       @Param("low") java.math.BigDecimal low, @Param("close") java.math.BigDecimal close,
                       @Param("volume") long volume, @Param("adjClose") java.math.BigDecimal adjClose);

    /**
     * Finds the most recent daily candle for a stock.
     *
     * @param symbol the stock symbol
     * @return optional containing the most recent candle
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' ORDER BY c.date DESC LIMIT 1")
    Optional<OhlcvCandleEntity> findLatestBySymbol(@Param("symbol") String symbol);

    /**
     * Finds all daily candles for a stock within a date range, ordered by date descending.
     *
     * @param symbol the stock symbol
     * @param startDate the start date (inclusive)
     * @param endDate the end date (inclusive)
     * @param pageable pagination
     * @return list of candles
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' AND c.date BETWEEN :startDate AND :endDate ORDER BY c.date DESC")
    List<OhlcvCandleEntity> findBySymbolAndDateRange(
        @Param("symbol") String symbol,
        @Param("startDate") LocalDate startDate,
        @Param("endDate") LocalDate endDate,
        Pageable pageable
    );

    /**
     * Finds the last N daily candles for a stock.
     *
     * @param symbol the stock symbol
     * @param count the number of candles to retrieve
     * @return list of candles
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' ORDER BY c.date DESC")
    List<OhlcvCandleEntity> findTopBySymbolOrderByDateDesc(
        @Param("symbol") String symbol,
        Pageable pageable
    );

    /**
     * Finds all daily candles for a stock, ordered by date descending.
     *
     * @param symbol the stock symbol
     * @return list of candles
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' ORDER BY c.date DESC")
    List<OhlcvCandleEntity> findAllBySymbolOrderByDateDesc(String symbol);

    /** Finds the unique daily candle for a symbol on a trading date. */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' AND c.date = :date")
    Optional<OhlcvCandleEntity> findBySymbolAndDate(String symbol, LocalDate date);

    /**
     * Finds all unique symbols that have daily candle data.
     *
     * @return list of unique symbols
     */
    @Query("SELECT DISTINCT c.symbol FROM OhlcvCandleEntity c WHERE c.timeframe = '" + DAILY_INTERVAL + "' ORDER BY c.symbol")
    List<String> findAllDistinctSymbols();

    /**
     * Checks if a daily candle exists for a stock on a specific date.
     *
     * @param symbol the stock symbol
     * @param date the date
     * @return true if the candle exists
     */
    @Query("SELECT COUNT(c) > 0 FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' AND c.date = :date")
    boolean existsBySymbolAndDate(
        @Param("symbol") String symbol,
        @Param("date") LocalDate date
    );

    /**
     * Finds the earliest daily candle for a stock.
     *
     * @return optional containing the earliest candle
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' ORDER BY c.date ASC LIMIT 1")
    Optional<OhlcvCandleEntity> findEarliestBySymbol(@Param("symbol") String symbol);

    /**
     * Counts the number of daily candles for a stock.
     *
     * @param symbol the stock symbol
     * @return the count
     */
    @Query("SELECT COUNT(c) FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "'")
    long countBySymbol(@Param("symbol") String symbol);

    /**
     * Deletes all candles for a stock, across every interval.
     *
     * @param symbol the stock symbol
     */
    void deleteBySymbol(String symbol);

    @Modifying
    @Query("DELETE FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' AND c.date = :date")
    int deleteBySymbolAndDate(@Param("symbol") String symbol, @Param("date") LocalDate date);

    /**
     * Finds daily candles by date range across all symbols.
     *
     * @param startDate the start date (inclusive)
     * @param endDate the end date (inclusive)
     * @param pageable pagination
     * @return list of candles
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.timeframe = '" + DAILY_INTERVAL + "' AND c.date BETWEEN :startDate AND :endDate ORDER BY c.date DESC, c.symbol ASC")
    List<OhlcvCandleEntity> findByDateRange(
        @Param("startDate") LocalDate startDate,
        @Param("endDate") LocalDate endDate,
        Pageable pageable
    );

    /** Finds a bounded page of daily candles for a selected set of symbols. */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol IN :symbols "
        + "AND c.timeframe = '" + DAILY_INTERVAL + "' "
        + "AND c.date BETWEEN :startDate AND :endDate ORDER BY c.date ASC, c.symbol ASC")
    List<OhlcvCandleEntity> findBySymbolsAndDateRange(
        @Param("symbols") List<String> symbols,
        @Param("startDate") LocalDate startDate,
        @Param("endDate") LocalDate endDate,
        Pageable pageable
    );

    /**
     * Finds the latest daily candle for a symbol before a given date/time.
     *
     * @param symbol the trading symbol
     * @param before the cutoff date/time
     * @return optional containing the latest candle before the cutoff
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' AND c.date < :before ORDER BY c.date DESC LIMIT 1")
    Optional<OhlcvCandleEntity> findLatestBySymbolBeforeDate(
        @Param("symbol") String symbol,
        @Param("before") LocalDate before
    );

    /**
     * Finds the Nth trading day candle after a given date.
     * Useful for computing returns over evaluation windows (1d, 5d, 21d).
     *
     * @param symbol the stock symbol
     * @param after the reference date (candles must be after this date)
     * @param n the Nth trading day (1 = next trading day, 5 = 5 trading days later)
     * @return optional containing the Nth candle
     */
    @Query(value = "SELECT * FROM ohlcv_candles WHERE symbol = :symbol AND timeframe = '" + DAILY_INTERVAL + "' AND date > :after ORDER BY date ASC LIMIT 1 OFFSET :n", nativeQuery = true)
    Optional<OhlcvCandleEntity> findNthBySymbolAndDateAfterOrderByDateAsc(
        @Param("symbol") String symbol,
        @Param("after") LocalDate after,
        @Param("n") int n
    );

    /**
     * Finds the latest daily candle for a symbol on or after a given date.
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = '" + DAILY_INTERVAL + "' AND c.date >= :date ORDER BY c.date ASC LIMIT 1")
    Optional<OhlcvCandleEntity> findFirstBySymbolAndDateAfterOrderByDateAsc(
        @Param("symbol") String symbol,
        @Param("date") LocalDate date
    );

    /**
     * Finds up to N daily candles for a symbol before a given date, in ascending order.
     * Used for computing SMA-200 leading up to a reference date.
     */
    @Query(value = "SELECT * FROM ohlcv_candles WHERE symbol = :symbol AND timeframe = '" + DAILY_INTERVAL + "' AND date <= :before ORDER BY date DESC LIMIT :n", nativeQuery = true)
    List<OhlcvCandleEntity> findLastNBySymbolBeforeDateAsc(
        @Param("symbol") String symbol,
        @Param("before") LocalDate before,
        @Param("n") int n
    );

    // -----------------------------------------------------------------------
    // Intraday (interval-aware) queries
    // -----------------------------------------------------------------------

    /**
     * Finds candles of a specific interval for a stock within a date range,
     * ordered by date and bar start time ascending.
     *
     * @param symbol the stock symbol
     * @param interval the candle interval
     * @param startDate the start date (inclusive)
     * @param endDate the end date (inclusive)
     * @param pageable pagination
     * @return list of candles
     */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = :interval "
        + "AND c.date BETWEEN :startDate AND :endDate ORDER BY c.date ASC, c.barTime ASC")
    List<OhlcvCandleEntity> findBySymbolAndIntervalAndDateRange(
        @Param("symbol") String symbol,
        @Param("interval") String interval,
        @Param("startDate") LocalDate startDate,
        @Param("endDate") LocalDate endDate,
        Pageable pageable
    );

    /** Finds the latest stored candle of a specific interval for a stock. */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = :interval "
        + "ORDER BY c.date DESC, c.barTime DESC LIMIT 1")
    Optional<OhlcvCandleEntity> findLatestBySymbolAndInterval(
        @Param("symbol") String symbol,
        @Param("interval") String interval
    );

    /** Finds the unique candle of a specific interval for a symbol on a trading date and bar start time. */
    @Query("SELECT c FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = :interval "
        + "AND c.date = :date AND c.barTime = :barTime")
    Optional<OhlcvCandleEntity> findBySymbolAndIntervalAndDateAndBarTime(
        @Param("symbol") String symbol,
        @Param("interval") String interval,
        @Param("date") LocalDate date,
        @Param("barTime") LocalTime barTime
    );

    /** Counts the candles of a specific interval for a stock. */
    @Query("SELECT COUNT(c) FROM OhlcvCandleEntity c WHERE c.symbol = :symbol AND c.timeframe = :interval")
    long countBySymbolAndInterval(
        @Param("symbol") String symbol,
        @Param("interval") String interval
    );
}
