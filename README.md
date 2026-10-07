# Task API

A production-style REST API in Go: JWT authentication, PostgreSQL, Docker, OpenAPI docs and tests.

**Stack:** Go 1.26 · Gin · PostgreSQL (pgx) · JWT · bcrypt · Docker Compose · Swagger UI

## Features

- Registration and login with bcrypt-hashed passwords and JWT tokens
- Full CRUD for tasks with partial updates (`PATCH`), filtering and pagination
- Strict per-user data isolation, enforced in every query
- Input validation with clear JSON errors
- Interactive API docs at `/docs` (OpenAPI 3)
- Structured JSON logging, graceful shutdown, HTTP timeouts
- Storage behind an interface: PostgreSQL in production, in-memory for local runs and tests
- Small, non-root distroless Docker image

## Quick start

With Docker (API + PostgreSQL):

```bash
docker compose up --build
```

Without Docker (in-memory storage):

```bash
go run ./cmd/api
```

Then open http://localhost:8080/docs.

## Configuration

| Variable       | Default                | Description                                  |
|----------------|------------------------|----------------------------------------------|
| `PORT`         | `8080`                 | HTTP port                                    |
| `DATABASE_URL` | empty                  | PostgreSQL URL; empty means in-memory store  |
| `JWT_SECRET`   | `dev-secret-change-me` | HMAC secret for signing tokens               |

## Endpoints

| Method | Path                    | Auth | Description                                   |
|--------|-------------------------|------|-----------------------------------------------|
| POST   | `/api/v1/auth/register` | –    | Create an account, returns a token            |
| POST   | `/api/v1/auth/login`    | –    | Log in, returns a token                       |
| GET    | `/api/v1/tasks`         | JWT  | List tasks (`?done=true&limit=20&offset=0`)   |
| POST   | `/api/v1/tasks`         | JWT  | Create a task                                 |
| GET    | `/api/v1/tasks/{id}`    | JWT  | Get a task                                    |
| PATCH  | `/api/v1/tasks/{id}`    | JWT  | Update any of `title`, `description`, `done`  |
| DELETE | `/api/v1/tasks/{id}`    | JWT  | Delete a task                                 |
| GET    | `/healthz`              | –    | Health check                                  |

## Example

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"me@example.com","password":"password123"}' | jq -r .token)

curl -X POST localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"Ship the MVP"}'

curl "localhost:8080/api/v1/tasks?done=false" -H "Authorization: Bearer $TOKEN"
```

## Project layout

```
cmd/api/            entry point: config, wiring, graceful shutdown
api/                OpenAPI spec and Swagger UI page (embedded into the binary)
internal/auth/      password hashing and JWT
internal/handler/   HTTP routes, middleware, validation, tests
internal/store/     Store interface, PostgreSQL and in-memory implementations
internal/model/     domain types
```

## Tests

```bash
go test -race ./...
```
