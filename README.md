# gofro

CLI scaffolding tool for Go projects. Generates a production-ready project structure — HTTP server, config, Docker Compose, and optional databases — all wired up and ready to `go run`.

Inspired by [logiflow](https://github.com/anxi0uz/logiflow).

[Русский](README_RU.md)

---

## Install

```sh
go install github.com/anxi0uz/gofro@latest
```

## Usage

```sh
gofro new <project-name> [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--postgres` | PostgreSQL: pgxpool connection, goose migrations, generic storage layer |
| `--redis` | Redis: go-redis/v9 client |
| `--prometheus` | Prometheus: scrape config in `configs/prometheus.yml` |
| `--grafana` | Grafana in Compose (automatically enables `--prometheus`) |
| `--github <nick>` | Module path → `github.com/<nick>/<project>` |
| `--module <path>` | Full custom module path for `go mod init` |
| `--git` | Run `git init` in the generated project |

### Examples

```sh
# Minimal API
gofro new myapi --github johndoe

# With databases
gofro new myapi --postgres --redis --github johndoe --git

# Full observability stack
gofro new myapi --postgres --redis --prometheus --grafana --github johndoe --git
```

## Generated structure

```
myapi/
├── cmd/
│   └── main.go                    # Thin entry point — init deps, call handler.NewServer().Run()
├── configs/
│   ├── config.toml                # App config (koanf, TOML + env vars)
│   └── prometheus.yml             # Only with --prometheus
├── internal/
│   ├── api/
│   │   ├── api.swagger.yaml       # OpenAPI 3.0 spec — define your endpoints here
│   │   ├── gen.go                 # //go:generate oapi-codegen directive
│   │   └── oapi-codegen.yaml      # Codegen config (chi-server + models)
│   ├── config/
│   │   └── config.go              # Config structs, koanf loader, DSN helpers
│   ├── database/
│   │   ├── postgres.go            # Only with --postgres: pgxpool + goose runner
│   │   └── redis.go               # Only with --redis: go-redis client
│   └── handler/
│       └── server_impl.go         # Server struct, JSON(), Routes(), Run()
├── pkg/
│   └── storage/
│       └── storage.go             # Only with --postgres: generic CRUD (GetAll/GetOne/Create/Update/Delete)
├── migrations/                    # Only with --postgres: SQL migration files go here
├── docker-compose.yml             # Only the services you selected
├── Dockerfile                     # Multi-stage Alpine build
├── .env                           # Env vars template (gitignored)
├── .gitignore
└── Makefile                       # build / run / generate / lint / clean
```

## Stack

| Concern | Library |
|---|---|
| HTTP router | [chi v5](https://github.com/go-chi/chi) |
| CORS | [go-chi/cors](https://github.com/go-chi/cors) |
| Config | [koanf v2](https://github.com/knadh/koanf) |
| Env loading | [godotenv](https://github.com/joho/godotenv) |
| Logging | `log/slog` + [devslog](https://github.com/golang-cz/devslog) |
| PostgreSQL | [pgx/v5](https://github.com/jackc/pgx) |
| SQL builder | [go-sqlbuilder](https://github.com/huandu/go-sqlbuilder) |
| Migrations | [goose v3](https://github.com/pressly/goose) |
| Redis | [go-redis/v9](https://github.com/redis/go-redis) |
| API codegen | [oapi-codegen v2](https://github.com/oapi-codegen/oapi-codegen) |

## Workflow after generation

### 1. Start infrastructure

```sh
cd myapi
docker compose up -d
```

### 2. Define your API in swagger

Edit `internal/api/api.swagger.yaml`, then run:

```sh
# Install oapi-codegen once
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

make generate
```

This generates `internal/api/api.gen.go` with typed models and the server interface.

### 3. Implement the interface

In `internal/handler/server_impl.go`, uncomment the router line and implement the generated methods:

```go
// Routes() — uncomment this line:
api.HandlerFromMux(s, r)

// Then add your handlers on the Server struct:
func (s *Server) GetUserById(w http.ResponseWriter, r *http.Request, id string) {
    user, err := storage.GetOne[User](r.Context(), s.db, "users",
        func(sb *sqlbuilder.SelectBuilder) { sb.Where(sb.Equal("id", id)) },
    )
    if err != nil {
        s.JSON(w, r, http.StatusNotFound, "error", map[string]string{"message": "not found"})
        return
    }
    s.JSON(w, r, http.StatusOK, "user", user)
}
```

### 4. JSON response format

`s.JSON(w, r, status, respType, payload)` always returns:

```json
{
    "request_id": "cv4sm3d0v9guj8u1c5o0",
    "status": 200,
    "success": true,
    "data": {
        "user": { "id": "...", "name": "..." }
    }
}
```

### 5. Generic storage (--postgres)

All functions work with any struct via generics. Fields tagged `db:"-"` are skipped on insert; fields tagged `immutable` are skipped on update.

```go
// SELECT all with optional filter
users, err := storage.GetAll[User](ctx, "users", s.db,
    func(sb *sqlbuilder.SelectBuilder) {
        sb.Where(sb.Equal("active", true))
        sb.Limit(20)
    },
)

// SELECT one
user, err := storage.GetOne[User](ctx, s.db, "users",
    func(sb *sqlbuilder.SelectBuilder) {
        sb.Where(sb.Equal("id", id))
    },
)
if errors.Is(err, storage.ErrNotFound) { ... }

// INSERT
err = storage.Create(ctx, "users", newUser, s.db)

// UPDATE (skips immutable fields like created_at)
err = storage.Update(ctx, "users", user, s.db,
    func(sb *sqlbuilder.UpdateBuilder) {
        sb.Where(sb.Equal("id", user.ID))
    },
)

// DELETE
err = storage.Delete[User](ctx, "users", s.db,
    func(sb *sqlbuilder.DeleteBuilder) {
        sb.Where(sb.Equal("id", id))
    },
)
```

### 6. Migrations

Put `.sql` files in `migrations/`. goose runs them automatically on startup.

```sql
-- migrations/00001_create_users.sql
-- +goose Up
CREATE TABLE users (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

-- +goose Down
DROP TABLE users;
```

## License

MIT
