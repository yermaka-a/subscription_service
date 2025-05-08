FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/subscription_service cmd/subpub/main.go

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /bin/subscription_service /app/subscription_service
ENTRYPOINT ["/app/subscription_service"]
EXPOSE 5000 