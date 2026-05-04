module github.com/mehrnazm/webhookx/go/ingestion

go 1.26.2

require (
	github.com/lib/pq v1.12.3
	github.com/mehrnazm/webhookx/go/util v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.19.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	go.uber.org/atomic v1.11.0 // indirect
)

replace github.com/mehrnazm/webhookx/go/util => ../util
