FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go/util ./go/util
COPY go/ingestion ./go/ingestion
WORKDIR /app/go/ingestion

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o ingestion .

FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/go/ingestion/ingestion /ingestion
EXPOSE 8080
ENTRYPOINT ["/ingestion"]
