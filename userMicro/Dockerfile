FROM golang:1.25-alpine AS builder-debug
WORKDIR /app
RUN apk add --no-cache git
COPY go.work go.work.sum ./
COPY common/go.mod ./common/
COPY authMicro/go.mod authMicro/go.sum ./authMicro/
COPY authMicro/grpcApi/go.mod ./authMicro/grpcApi/
COPY userMicro/go.mod userMicro/go.sum ./userMicro/
WORKDIR /app/userMicro
RUN go mod download
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY common/ /app/common/
COPY authMicro/grpcApi/ /app/authMicro/grpcApi/
COPY userMicro/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags "all=-N -l" -o user-app ./cmd/app

FROM golang:1.25-alpine AS builder-release
WORKDIR /app
RUN apk add --no-cache git
COPY go.work go.work.sum ./
COPY common/go.mod ./common/
COPY authMicro/go.mod authMicro/go.sum ./authMicro/
COPY authMicro/grpcApi/go.mod ./authMicro/grpcApi/
COPY userMicro/go.mod userMicro/go.sum ./userMicro/
WORKDIR /app/userMicro
RUN go mod download
COPY common/ /app/common/
COPY authMicro/grpcApi/ /app/authMicro/grpcApi/
COPY userMicro/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o user-app ./cmd/app

FROM alpine:3.19 AS run-debug
WORKDIR /app
COPY --from=builder-debug /app/userMicro/user-app .
COPY --from=builder-debug /app/userMicro/internal/infrastructure/adapter/repository/database/migrations /migrations
COPY --from=builder-debug /app/userMicro/locales ./locales
COPY --from=builder-debug /go/bin/dlv /usr/local/bin/dlv
EXPOSE 40001
EXPOSE 80
CMD dlv exec ./user-app --headless --listen=:40001 --api-version=2 --accept-multiclient --continue --log

FROM alpine:3.19 AS run-release
WORKDIR /app
COPY --from=builder-release /app/userMicro/user-app .
COPY --from=builder-release /app/userMicro/internal/infrastructure/adapter/repository/database/migrations /migrations
COPY --from=builder-release /app/userMicro/locales ./locales
EXPOSE 80
CMD ["./user-app"]