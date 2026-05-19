FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go/util ./go/util
COPY go/data-api ./go/data-api

WORKDIR /app/go/data-api

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o data-api .

FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/go/data-api/data-api /data-api
COPY postgres/migrations /postgres/migrations
EXPOSE 8081
ENTRYPOINT ["/data-api"]
