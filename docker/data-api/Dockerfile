FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go/data-api/go.mod go/data-api/go.sum ./
RUN go mod download
COPY go/data-api/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /data-api .

FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /data-api /data-api
EXPOSE 8081
ENTRYPOINT ["/data-api"]
