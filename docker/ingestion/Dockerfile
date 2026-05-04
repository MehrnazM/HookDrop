FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go/ingestion/go.mod go/ingestion/go.sum ./
RUN go mod download
COPY go/ingestion/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ingestion .

FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /ingestion /ingestion
EXPOSE 8080
ENTRYPOINT ["/ingestion"]
