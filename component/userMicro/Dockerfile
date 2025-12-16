FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-app ./cmd/app

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/auth-app .

COPY --from=builder /app/internal/repository/database/migrations /migrations

EXPOSE 8080

CMD ["./auth-app"]