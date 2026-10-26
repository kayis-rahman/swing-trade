module github.com/kayis-rahman/swing-trade/go/strategy

go 1.27.0

replace github.com/kayis-rahman/swing-trade/go/core => ../core

replace github.com/kayis-rahman/swing-trade/go/data => ../data

replace github.com/kayis-rahman/swing-trade/go/llm => ../llm

require (
	github.com/kayis-rahman/swing-trade/go/core v0.0.0-00010101000000-000000000000
	github.com/kayis-rahman/swing-trade/go/data v0.0.0-00010101000000-000000000000
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.2 // indirect
	golang.org/x/text v0.41.0 // indirect
)
