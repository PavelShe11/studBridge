# studBridge

Backend системы аутентификации и управления аккаунтами, построенная на микросервисной архитектуре с использованием Go. [`ТЗ`](docs/ТЗ_Auth.md)

## Architecture

```
                          ┌─────────────────────────────────────┐
                          │           Client (HTTP/REST)        │
                          └──────────────────┬──────────────────┘
                                             │
                          ┌──────────────────▼──────────────────┐
                          │         Traefik API Gateway         │
                          │         localhost:80 / :8088        │
                          └──────────────────┬──────────────────┘
                                             │  /auth/v1/*
                          ┌──────────────────▼──────────────────┐
                          │           Auth Service              │
                          │     REST (Echo) + gRPC client       │
                          │                                     │
                          │  • Registration flow (2-step)       │
                          │  • Login flow (2-step OTP)          │
                          │  • JWT refresh tokens               │
                          └──────┬─────────────────┬────────────┘
                                 │ gRPC            │ SQL
                     ┌───────────▼────────┐  ┌─────▼──────────┐
                     │    User Service    │  │    Auth DB     │
                     │  (gRPC server)     │  │  PostgreSQL    │
                     │                    │  │  port: 5433    │
                     │ • validateAccount  │  └────────────────┘
                     │ • createAccount    │
                     │ • getAccount       │
                     │ • updateAccount    │
                     └─────────┬──────────┘
                               │ SQL
                     ┌─────────▼──────────┐
                     │     User DB        │
                     │   PostgreSQL       │
                     │   port: 5434       │
                     └────────────────────┘
```

**Потоки данных:**

- **Регистрация:** Client → Auth (создаёт сессию + код) → Client вводит код → Auth проверяет → gRPC User (создаёт
  аккаунт)
- **Вход:** Client → Auth (находит аккаунт + отправляет OTP) → Client вводит код → Auth → JWT tokens
- **Refresh:** Client → Auth (проверяет refresh token) → новая пара токенов

## Tech Stack

| Компонент        | Технология                    |
|------------------|-------------------------------|
| Language         | Go 1.25.5                     |
| REST Framework   | Echo v4                       |
| Inter-service    | gRPC + Protocol Buffers       |
| API Gateway      | Traefik 3.4                   |
| Database         | PostgreSQL 16-Alpine          |
| Migrations       | golang-migrate                |
| Auth             | JWT (access + refresh tokens) |
| Logging          | Uber Zap                      |
| Localization     | go-i18n (en, ru)              |
| Testing          | Testcontainers, Mockery       |
| API Docs         | Swagger (swaggo)              |
| Containerization | Docker                |

## Quick Start

```bash
# Настройте переменные, заменив те что по умолчанию
.env

# Для поднятия всех сервисов
docker compose up -d --build

# Для остановки
docker compose down
```

**Адреса:**

- Auth API: `http://localhost/auth/v1/`
- Swagger UI: `http://localhost/auth/v1/swagger/index.html`
- Health check: `http://localhost/auth/v1/health`
- Traefik dashboard: `http://localhost:8088` (admin / P@ssw0rd)

**Отладка (Delve):**

```bash
docker compose -f docker-compose.yml -f docker-compose.debug.yml up -d --build
# Auth service: dlv connect localhost:40000
# User service: dlv connect localhost:40001
```

## API Overview

| Метод  | Путь                                 | Описание                                  |
|--------|--------------------------------------|-------------------------------------------|
| `POST` | `/auth/v1/registration`              | Начать регистрацию — отправляет OTP-код   |
| `POST` | `/auth/v1/registration/confirmEmail` | Подтвердить email — завершает регистрацию |
| `POST` | `/auth/v1/login/sendCodeEmail`       | Начать вход — отправляет OTP-код          |
| `POST` | `/auth/v1/login/confirmEmail`        | Подтвердить вход — возвращает JWT токены  |
| `POST` | `/auth/v1/refreshToken`              | Обновить пару токенов по refresh token    |
| `GET`  | `/auth/v1/health`                    | Health check                              |

Полная интерактивная документация: **Swagger UI** — `http://localhost/auth/v1/swagger/index.html`

## Development

### Регенерация Swagger документации

```bash
cd authMicro
swag init -g internal/infrastructure/inbound/rest/router.go --parseDependency --parseInternal
```

Генерирует файлы в `authMicro/docs/`. Запускать после изменения аннотаций в хендлерах или `router.go`.

## Testing

Тесты покрывают auth service: unit, integration (Testcontainers), e2e.

Подробная документация по тестам: [`authMicro/TEST_README.md`](authMicro/TEST_README.md)

```bash
# Запустить все тесты auth service
cd authMicro
go test ./...

# Запустить с подробным выводом
go test -v ./...

# Только unit тесты (без Testcontainers)
go test -v -run TestUnit ./...

# Только интеграционные тесты
go test -v -run TestIntegration ./...
```

> Для интеграционных тестов требуется запущенный Docker.

## Project Structure

```
studBridge/
├── authMicro/                  # Auth service (REST API + gRPC client)
│   ├── cmd/app/main.go         # Точка входа, DI-сборка приложения
│   ├── grpcApi/                # Shared gRPC generated code + proto files
│   │   └── proto/              # .proto definitions (account_service, error)
│   ├── internal/
│   │   ├── config/             # Env-based конфигурация
│   │   ├── entity/             # Доменные сущности (чистые Go-структуры)
│   │   ├── port/               # Интерфейсы репозиториев и провайдеров
│   │   ├── service/            # Бизнес-логика (registration, login, token)
│   │   ├── usecase/            # Оркестрация (AuthenticateUser)
│   │   └── infrastructure/
│   │       ├── inbound/rest/   # Echo HTTP handlers, router, middleware
│   │       └── outbound/       # gRPC adapter, DB repositories
│   ├── locales/                # i18n файлы (en.toml, ru.toml)
│   └── TEST_README.md          # Документация по тестированию
│
├── userMicro/                  # User service (gRPC server only)
│   ├── cmd/app/main.go         # Точка входа
│   └── internal/               # Та же Clean Architecture
│
├── common/                     # Shared библиотеки (logger, translator, errors)
├── dynamic/                    # Traefik dynamic config (routers, services)
├── docker-compose.yml          # Production compose
├── docker-compose.debug.yml    # Debug compose (Delve ports)
├── go.work                     # Go workspace (multi-module)
└── traefic.yml                 # Traefik static config
```

### Архитектурные слои (Clean Architecture)

```
HTTP Request
    │
    ▼
Handler (inbound/rest/handler)
    │  DTO binding & response mapping
    ▼
UseCase (usecase/)
    │  Orchestration, transactions
    ▼
Service (service/)
    │  Business logic
    ▼
Port interface (port/)
    │  Repository / Provider contracts
    ▼
Adapter (outbound/)
    │  DB queries, gRPC calls
    ▼
PostgreSQL / gRPC
```
