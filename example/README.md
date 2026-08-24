# API Observer Security Test Service

A deliberately vulnerable REST API for controlled API Observer evaluation against:

- A01 – Broken Access Control
- A02 – Security Misconfiguration
- A05 – Injection
- A07 – Authentication Failures
- A09 – Security Logging and Alerting Failures

The service uses Go's `net/http` package and SQLite via `modernc.org/sqlite`.

## Project layout

```text
.
├── api/
│   └── openapi.yaml
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── audit.go
│   │   ├── auth.go
│   │   ├── projects.go
│   │   ├── security.go
│   │   ├── tasks.go
│   │   ├── types.go
│   │   ├── users.go
│   │   └── util.go
│   ├── config/
│   │   └── config.go
│   └── database/
│       ├── database.go
│       └── migrations/
│           └── 001_init.sql
├── go.mod
└── README.md
```

`cmd/server` is the executable composition root. `internal/app` contains the HTTP application, while `internal/database` owns SQLite initialization, migrations, and deterministic seed data.

## Run

```bash
go mod tidy
go run ./cmd/server
```

Environment variables:

```bash
TEST_SERVICE_ADDR=:8080
TEST_SERVICE_DB=./test.db
```

## Seed accounts

| User | Password | Role |
|---|---|---|
| `alice@example.com` | `password123` | standard |
| `bob@example.com` | `password123` | standard |
| `admin@example.com` | `adminpass123` | admin |

Alice owns project `10000000-0000-0000-0000-000000000001` and Bob owns project `10000000-0000-0000-0000-000000000002`.

## OpenAPI

The contract is located at:

```text
api/openapi.yaml
```

## Intentional security behavior

This application is intentionally insecure and should only be used in a controlled test environment.

The Broken Access Control scenarios deliberately omit selected project ownership and role-management checks. The Security Misconfiguration scenarios expose synthetic debug/configuration information. The Injection scenarios include an intentionally unsafe SQLite user-search query but never execute operating-system commands or read arbitrary host files. Authentication deliberately lacks throttling/lockout. Application audit logging is intentionally incomplete so it can be compared with API Observer's independent visibility.
