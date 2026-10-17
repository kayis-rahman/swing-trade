# api

Go module `swingtrade/api`, corresponding to Gradle module `backend/api`; the executable lives in `cmd/swingtrade-api`. Owns HTTP controllers, orchestration, DTOs, config wiring, schedulers, middleware and API metrics. May import all six supporting modules.

Phase 1 exposes infrastructure health and metrics routes only. It is not routed to production traffic.
