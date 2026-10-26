module github.com/kayis-rahman/swing-trade/go/data

go 1.27.0

require (
	github.com/jackc/pgx/v5 v5.9.2
	github.com/kayis-rahman/swing-trade/go/core v0.0.0-00010101000000-000000000000
	github.com/pashagolub/pgxmock/v4 v4.9.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/kayis-rahman/swing-trade/go/core => ../core
