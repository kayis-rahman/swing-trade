module github.com/kayis-rahman/swing-trade/go/api

go 1.27.0

require (
	github.com/kayis-rahman/swing-trade/go/core v0.0.0
	github.com/kayis-rahman/swing-trade/go/data v0.0.0
	github.com/prometheus/client_golang v1.22.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace github.com/kayis-rahman/swing-trade/go/broker => ../broker

replace github.com/kayis-rahman/swing-trade/go/core => ../core

replace github.com/kayis-rahman/swing-trade/go/data => ../data

replace github.com/kayis-rahman/swing-trade/go/gpuhub => ../gpuhub

replace github.com/kayis-rahman/swing-trade/go/llm => ../llm

replace github.com/kayis-rahman/swing-trade/go/strategy => ../strategy
