# 🏦 GoBank

A production-grade banking API built with Go, gRPC, and gRPC-Gateway.

Supports user registration, email verification, PASETO-based authentication with access/refresh tokens, multi-currency accounts, and atomic money transfers with **deadlock-safe** transactions.

---

## Table of Contents

- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Running Locally](#running-locally)
- [Running with Docker Compose](#running-with-docker-compose)
- [Running Tests](#running-tests)
- [API Documentation](#api-documentation)
- [Project Structure](#project-structure)
- [CI/CD](#cicd)
- [Deployment (AWS EKS)](#deployment-aws-eks)

---

## Architecture

```
HTTP Client
	│
	▼
gRPC-Gateway (HTTP :8080)          ← REST + Swagger UI
	│
	▼
gRPC Server (:9090)                ← auth + logging interceptors
	│
	├── PostgreSQL (pgxpool)        ← primary store
	├── Redis (Asynq)               ← background task queue
	└── Gmail SMTP                  ← email verification
```

Every HTTP request is transcoded by gRPC-Gateway into a gRPC call, meaning **all auth and logging interceptors run for both REST and gRPC traffic**.

---

## Tech Stack

| Layer            | Technology                 |
| ---------------- | -------------------------- |
| Language         | Go 1.26.1                  |
| API              | gRPC + gRPC-Gateway (REST) |
| Auth             | PASETO v2 (symmetric)      |
| Database         | PostgreSQL 16 via `pgx/v5` |
| Migrations       | `golang-migrate`           |
| Query gen        | `sqlc`                     |
| Task queue       | Asynq (Redis-backed)       |
| Email            | Gmail SMTP via `go-mail`   |
| Logging          | Uber Zap (structured JSON) |
| Containerization | Docker + Docker Compose    |
| CI/CD            | GitHub Actions             |
| Infra            | AWS ECR + EKS              |

---

## Features

- **User management** — register, login, update profile, email verification
- **PASETO tokens** — short-lived access tokens + long-lived refresh tokens with session store
- **Multi-currency accounts** — USD, EUR, EGP; one account per currency per user
- **Atomic transfers** — deadlock-safe transaction ordering; balances stored as integers (cents) to avoid floating-point issues
- **Activity feed** — enriched entry listing with counterpart account info, currency, and transfer linkage via `ListActivityEntries`
- **Account lookup** — lightweight `LookUpAccount` endpoint for transfer recipient validation (returns owner + currency, no balance)
- **Background jobs** — email verification dispatched asynchronously via Redis/Asynq
- **Swagger UI** — served at `/swagger/` with the OpenAPI spec embedded in the binary
- **Structured logging** — per-request correlation IDs, user context, gRPC codes, latency
- **CORS** — configurable allowed origins for local dev and Vercel-hosted frontends

---

## Prerequisites

| Tool                    | Version  | Install                               |
| ----------------------- | -------- | ------------------------------------- |
| Go                      | 1.26.1   | https://go.dev/dl                     |
| Docker + Docker Compose | latest   | https://docs.docker.com/get-docker    |
| `golang-migrate`        | v4.19.1+ | See below                             |
| `sqlc`                  | v1.30.0  | https://docs.sqlc.dev                 |
| `protoc` + plugins      | latest   | Only needed to regenerate proto files |

**Install golang-migrate (macOS/Linux):**

```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/bin/
```

---

## Environment Setup

The project uses two separate env files for different purposes.

### `app.env` — Go application config

Create `app.env` in the project root. This file is loaded by Viper at startup and is also used inside the Docker container.

```env
# Server
PORT=8080
SERVER_ADDRESS=0.0.0.0
GRPC_SERVER_PORT=9090
ENVIRONMENT=development

# Database (used by the running app)
DB_URL=postgresql://root:password@localhost:5432/bank?sslmode=disable

# Database (used only by tests — can point to same DB or a separate test DB)
TESTING_DB_URL=postgresql://root:password@localhost:5432/bank?sslmode=disable

# Auth
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h

# Redis
REDIS_ADDRESS=localhost:6379

# Public base URL (used in email verification links)
BASE_URL=http://localhost:8080

# Email (Gmail App Password — NOT your real Gmail password)
EMAIL_SENDER_NAME=GoBank
EMAIL_SENDER_ADDRESS=your-email@gmail.com
EMAIL_SENDER_PASSWORD=your-gmail-app-password
```

> **TOKEN_SYMMETRIC_KEY must be exactly 32 characters** (required by ChaCha20-Poly1305).

> **Gmail App Password**: Go to your Google Account → Security → 2-Step Verification → App passwords. Generate one for "Mail".

### `.env` — Docker Compose / Makefile config

Create `.env` in the project root. This is only used by Docker Compose and the Makefile targets that spin up local Postgres/Redis.

```env
POSTGRES_USER=root
POSTGRES_PASSWORD=password
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
BASE_URL=http://localhost:8080
```

Both files are already in `.gitignore` — **never commit them**.

---

## Running Locally

### 1. Create the Docker network

> ⚠️ **This step must come first.** The Postgres container is attached to `bank-network`, and the network must exist before you start it.

```bash
docker network create bank-network
```

### 2. Start dependencies

```bash
# Start Postgres (attached to bank-network)
make postgres

# Create the database
make createdb

# Start Redis
make redis
```

### 3. Run migrations

```bash
make migrateup
```

### 4. Start the server

```bash
make server
```

The server starts two listeners:

- HTTP gateway + Swagger UI → `http://localhost:8080`
- gRPC server → `localhost:9090`

### Useful Makefile targets

| Command                       | Description                                   |
| ----------------------------- | --------------------------------------------- |
| `make server`                 | Run the app                                   |
| `make test`                   | Run all tests (short mode, skips integration) |
| `make migrateup`              | Apply all pending migrations                  |
| `make migratedown`            | Roll back all migrations                      |
| `make migrateup1`             | Apply one migration                           |
| `make migratedown1`           | Roll back one migration                       |
| `make new_migration name=<n>` | Create a new migration file pair              |
| `make sqlc`                   | Regenerate db query code from SQL             |
| `make proto`                  | Regenerate protobuf Go files + OpenAPI spec   |
| `make evans`                  | Open Evans gRPC REPL                          |

---

## Running with Docker Compose

Docker Compose runs Postgres, Redis, and the API together. Background email jobs work out of the box.

```bash
# Make sure .env exists (see Environment Setup above)
docker-compose up --build
```

The API will be available at `http://localhost:8080` once the Postgres and Redis health checks pass and migrations complete.

To stop and remove containers:

```bash
docker-compose down
```

To also remove the Postgres volume (wipes data):

```bash
docker-compose down -v
```

---

## Running Tests

Tests require a running Postgres instance. The test suite reads `TESTING_DB_URL` from `app.env`.

```bash
# Run all tests (short mode — skips the real email integration test)
make test

# Run a specific package
go test -v ./db/sqlc/...
go test -v ./token/...
go test -v ./api/...
go test -v ./gapi/...

# Run the real email integration test (requires valid Gmail credentials in app.env)
go test -v -run TestSendRealEmail ./mail/
```

The CI pipeline (`.github/workflows/test.yml`) spins up a Postgres service container automatically — no manual setup needed for CI.

---

## API Documentation

Once the server is running, the Swagger UI is available at:

```
http://localhost:8080/swagger/
```

The raw OpenAPI 2.0 spec (useful for Postman or code generators) is at:

```
http://localhost:8080/swagger/doc.json
```

### Quick API overview

| Endpoint                | Method | Auth | Description                                    |
| ----------------------- | ------ | ---- | ---------------------------------------------- |
| `/v1/users`             | POST   | ❌   | Register a new user                            |
| `/v1/auth/login`        | POST   | ❌   | Login, get access + refresh tokens             |
| `/v1/auth/renew_access` | POST   | ❌   | Renew access token using refresh token         |
| `/v1/verify_email`      | GET    | ❌   | Verify email via link                          |
| `/v1/users`             | PATCH  | ✅   | Update your profile                            |
| `/v1/accounts`          | POST   | ✅   | Create a currency account                      |
| `/v1/accounts`          | GET    | ✅   | List your accounts (paginated)                 |
| `/v1/accounts/:id`      | GET    | ✅   | Get a specific account                         |
| `/v1/accounts/lookup`   | GET    | ✅   | Look up any account by ID (for transfers)      |
| `/v1/accounts/:id`      | PUT    | ✅   | Update account balance                         |
| `/v1/accounts/:id`      | DELETE | ✅   | Delete an account                              |
| `/v1/transfers`         | POST   | ✅   | Transfer funds between accounts                |
| `/v1/entries`           | GET    | ✅   | List activity entries with counterpart details |

All protected endpoints require `Authorization: Bearer <access_token>` in the header.

### Testing with Evans (gRPC REPL)

```bash
# Make sure the server is running first
make evans
```

---

## Project Structure

```
.
├── api/            # Gin HTTP server (legacy, kept for reference)
├── db/
│   ├── migration/  # SQL migration files (golang-migrate)
│   ├── query/      # Raw SQL queries (sqlc input)
│   └── sqlc/       # Auto-generated Go db code + transactions
├── gapi/           # gRPC handlers + auth/logging middleware
├── logger/         # Structured zap logger with HTTP + gRPC interceptors
├── mail/           # Gmail SMTP sender
├── pb/             # Auto-generated protobuf Go code
├── proto/          # .proto source files
├── token/          # PASETO + JWT maker implementations
├── util/           # Config, currencies, money conversion, random helpers
├── val/            # gRPC request validators
├── worker/         # Asynq task distributor + processor (background jobs)
├── doc/
│   ├── db.dbml     # Database schema (dbdocs.io)
│   └── swagger/    # Auto-generated OpenAPI spec
├── eks/            # Kubernetes deployment + service manifests
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── main.go
```

---

## CI/CD

### Test workflow (`.github/workflows/test.yml`)

Triggers on every push and pull request to `main`. It:

1. Spins up a Postgres service container
2. Installs `golang-migrate`
3. Runs all migrations against the test DB
4. Runs `go test -v -short -cover ./...`

**Required GitHub secrets:**

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`

### Deploy workflow (`.github/workflows/deploy.yml`)

Triggered manually (`workflow_dispatch`). It:

1. Authenticates to AWS
2. Pulls production secrets from AWS Secrets Manager (`gobank/prod`)
3. Builds and pushes the Docker image to Amazon ECR

**Required GitHub secrets:**

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

---

## Deployment (AWS EKS)

The `eks/` directory contains Kubernetes manifests for deploying to EKS.

```bash
# Apply deployment
kubectl apply -f eks/deployment.yaml

# Apply load balancer service
kubectl apply -f eks/service.yaml
```

Before applying, update the image tag in `eks/deployment.yaml` to the SHA of the image you pushed to ECR.

The service is exposed on port 80 → container port 8080 via an AWS LoadBalancer.

---

## License

GoBank is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
