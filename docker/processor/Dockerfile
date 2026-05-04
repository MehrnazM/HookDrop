FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go/processor/go.mod go/processor/go.sum ./
RUN go mod download
COPY go/processor/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /processor .

FROM alpine:3.22.4
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /processor /processor
ENTRYPOINT ["/processor"]
