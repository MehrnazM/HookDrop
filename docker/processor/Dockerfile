FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go/util ./go/util
COPY go/processor ./go/processor
WORKDIR /app/go/processor

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o processor .

FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/go/processor/processor /processor
ENTRYPOINT ["/processor"]
