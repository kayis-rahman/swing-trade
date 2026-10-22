# core

Go module `github.com/kayis-rahman/swing-trade/go/core`, corresponding to Gradle module `backend/core`. Owns domain values, ports, clock, config, decimal and logging primitives. Imports only the Go standard library.

Package layout mirrors `backend/core/src/main/java/com/swingtrade/domain/`; later domain ports belong in `domain/` and implementations remain in their owning modules.
