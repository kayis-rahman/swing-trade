# data

Go module `swingtrade/data`, corresponding to Gradle module `backend/data`. Owns PostgreSQL access, repositories, stores, market-data clients and ingestion services. May import `swingtrade/core` only.

Package layout mirrors the Java `entity`, `repository`, `store`, `client` and `service` roles. Database and provider calls do not enter `core` or `strategy`.
