FROM golang:1.25-alpine AS builder-debug
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags "all=-N -l" -o auth-app ./cmd/app

FROM golang:1.25-alpine AS builder-release
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-app ./cmd/app

FROM alpine:3.19 AS run-debug
WORKDIR /app
COPY --from=builder-debug /app/auth-app .
COPY --from=builder-debug /app/internal/infrastructure/adapter/repository/database/migrations /migrations
COPY --from=builder-debug /app/locales ./locales
COPY --from=builder-debug /go/bin/dlv /usr/local/bin/dlv
EXPOSE 40000
EXPOSE 80
CMD dlv exec ./auth-app --headless --listen=:40000 --api-version=2 --accept-multiclient --continue --log

FROM alpine:3.19 AS run-release
WORKDIR /app
COPY --from=builder-release /app/auth-app .
COPY --from=builder-release /app/internal/infrastructure/adapter/repository/database/migrations /migrations
COPY --from=builder-release /app/locales ./locales
EXPOSE 80
CMD ./auth-app
