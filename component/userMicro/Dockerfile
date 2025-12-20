FROM golang:1.25-alpine AS builder-debug
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags "all=-N -l" -o user-app ./cmd/app

FROM golang:1.25-alpine AS builder-release
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o user-app ./cmd/app

FROM alpine:3.19 AS run-debug
WORKDIR /app
COPY --from=builder-debug /app/user-app .
COPY --from=builder-debug /app/internal/repository/database/migrations /migrations
COPY --from=builder-debug /go/bin/dlv /usr/local/bin/dlv
EXPOSE 40001
EXPOSE 80
CMD dlv exec ./user-app --headless --listen=:40001 --api-version=2 --accept-multiclient --continue --log

FROM alpine:3.19 AS run-release
WORKDIR /app
COPY --from=builder-release /app/user-app .
COPY --from=builder-release /app/internal/repository/database/migrations /migrations
EXPOSE 80
CMD ["./user-app"]