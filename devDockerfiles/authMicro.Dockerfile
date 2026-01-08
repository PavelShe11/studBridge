FROM golang:1.25-alpine AS builder-debug
WORKDIR /workspace
RUN apk add --no-cache git
COPY ./common/go.mod common/go.sum ./common/
COPY ./authMicro/go.mod authMicro/go.sum ./authMicro/
COPY ./authMicro/grpcApi/go.mod ./authMicro/grpcApi/go.sum ./authMicro/grpcApi/
RUN go work init ./common ./authMicro ./authMicro/grpcApi
RUN go work sync
RUN go mod download
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY ./common/ /workspace/common/
COPY ./authMicro/ /workspace/authMicro/
COPY ./authMicro/grpcApi/ /workspace/authMicro/grpcApi/
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags "all=-N -l" -o auth-app ./authMicro/cmd/app

FROM alpine:3.19 AS run-debug
WORKDIR /app
COPY --from=builder-debug /workspace/auth-app .
COPY --from=builder-debug /workspace/authMicro/internal/infrastructure/adapter/repository/database/migrations /migrations
COPY --from=builder-debug /workspace/authMicro/locales ./locales
COPY --from=builder-debug /go/bin/dlv /usr/local/bin/dlv
EXPOSE 80
CMD dlv exec ./auth-app --headless --listen=:${DEBUG_PORT:-40000} --api-version=2 --accept-multiclient --continue --log