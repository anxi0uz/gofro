# gofro

CLI-инструмент для генерации Go-проектов. Создаёт готовую к продакшену структуру — HTTP-сервер, конфиг, Docker Compose и опциональные базы данных — всё подключено и готово к `go run`.

Вдохновлён [logiflow](https://github.com/anxi0uz/logiflow).

[English](README.md)

---

## Установка

```sh
go install github.com/anxi0uz/gofro@latest
```

`go install` кладёт бинарник в `$(go env GOPATH)/bin` (обычно `~/go/bin`). Убедись, что эта директория есть в `$PATH`.

### Добавление GOPATH/bin в PATH

**bash** — добавь в `~/.bashrc` или `~/.bash_profile`:
```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

**zsh** — добавь в `~/.zshrc`:
```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

**fish** — добавь в `~/.config/fish/config.fish`:
```fish
fish_add_path (go env GOPATH)/bin
```

После редактирования перезагрузи шелл (`source ~/.bashrc` / `source ~/.zshrc`) или открой новый терминал. Проверь:

```sh
gofro --help
gofro --version
```

## Использование

```sh
gofro new <имя-проекта> [флаги]
```

### Флаги

| Флаг | Описание |
|---|---|
| `--postgres` | PostgreSQL: pgxpool, goose-миграции, generic-слой хранилища |
| `--redis` | Redis: клиент go-redis/v9 |
| `--prometheus` | Prometheus: конфиг скрейпинга в `configs/prometheus.yml` |
| `--grafana` | Grafana в Compose (автоматически включает `--prometheus`) |
| `--github <ник>` | Путь модуля → `github.com/<ник>/<проект>` |
| `--module <путь>` | Полный кастомный путь модуля для `go mod init` |
| `--git` | Выполнить `git init` в сгенерированном проекте |

### Примеры

```sh
# Минимальный API
gofro new myapi --github johndoe

# С базами данных
gofro new myapi --postgres --redis --github johndoe --git

# Полный стек с мониторингом
gofro new myapi --postgres --redis --prometheus --grafana --github johndoe --git
```

## Структура генерируемого проекта

```
myapi/
├── cmd/
│   └── main.go                    # Точка входа — инициализация зависимостей, вызов handler.NewServer().Run()
├── configs/
│   ├── config.toml                # Конфиг приложения (koanf, TOML + переменные окружения)
│   └── prometheus.yml             # Только с флагом --prometheus
├── internal/
│   ├── api/
│   │   ├── api.swagger.yaml       # Спецификация OpenAPI 3.0 — описывай эндпоинты здесь
│   │   ├── gen.go                 # Директива //go:generate oapi-codegen
│   │   └── oapi-codegen.yaml      # Конфиг кодогенератора (chi-server + models)
│   ├── config/
│   │   └── config.go              # Структуры конфига, загрузчик koanf, хелперы DSN
│   ├── database/
│   │   ├── postgres.go            # Только с --postgres: pgxpool + запуск goose
│   │   └── redis.go               # Только с --redis: клиент go-redis
│   └── handler/
│       └── server_impl.go         # Структура Server, метод JSON(), Routes(), Run()
├── pkg/
│   └── storage/
│       └── storage.go             # Только с --postgres: generic CRUD (GetAll/GetOne/Create/Update/Delete)
├── migrations/                    # Только с --postgres: SQL-файлы миграций
├── docker-compose.yml             # Только выбранные сервисы
├── Dockerfile                     # Multi-stage сборка на Alpine
├── .env                           # Шаблон переменных окружения (добавлен в .gitignore)
├── .gitignore
└── Makefile                       # build / run / generate / lint / clean
```

## Стек

| Задача | Библиотека |
|---|---|
| HTTP-роутер | [chi v5](https://github.com/go-chi/chi) |
| CORS | [go-chi/cors](https://github.com/go-chi/cors) |
| Конфиг | [koanf v2](https://github.com/knadh/koanf) |
| Загрузка .env | [godotenv](https://github.com/joho/godotenv) |
| Логирование | `log/slog` + [devslog](https://github.com/golang-cz/devslog) |
| PostgreSQL | [pgx/v5](https://github.com/jackc/pgx) |
| SQL-билдер | [go-sqlbuilder](https://github.com/huandu/go-sqlbuilder) |
| Миграции | [goose v3](https://github.com/pressly/goose) |
| Redis | [go-redis/v9](https://github.com/redis/go-redis) |
| Кодогенерация API | [oapi-codegen v2](https://github.com/oapi-codegen/oapi-codegen) |

## Процесс работы после генерации

### 1. Запуск инфраструктуры

```sh
cd myapi
docker compose up -d
```

### 2. Описание API в swagger

Отредактируй `internal/api/api.swagger.yaml`, затем запусти:

```sh
# Установить oapi-codegen (один раз)
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

make generate
```

Это создаст `internal/api/api.gen.go` с типизированными моделями и интерфейсом сервера.

### 3. Реализация интерфейса

В `internal/handler/server_impl.go` раскомментируй строку роутера и добавь методы:

```go
// Routes() — раскомментировать:
api.HandlerFromMux(s, r)

// Реализовать методы на структуре Server:
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

### 4. Формат JSON-ответов

`s.JSON(w, r, status, respType, payload)` всегда возвращает:

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

### 5. Generic-хранилище (--postgres)

Все функции работают с любой структурой через дженерики. Поля с тегом `db:"-"` пропускаются при INSERT; поля с тегом `immutable` — при UPDATE.

```go
// SELECT всех с опциональным фильтром
users, err := storage.GetAll[User](ctx, "users", s.db,
    func(sb *sqlbuilder.SelectBuilder) {
        sb.Where(sb.Equal("active", true))
        sb.Limit(20)
    },
)

// SELECT одного
user, err := storage.GetOne[User](ctx, s.db, "users",
    func(sb *sqlbuilder.SelectBuilder) {
        sb.Where(sb.Equal("id", id))
    },
)
if errors.Is(err, storage.ErrNotFound) { ... }

// INSERT
err = storage.Create(ctx, "users", newUser, s.db)

// UPDATE (пропускает immutable поля, например created_at)
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

### 6. Миграции

Кладёшь `.sql`-файлы в `migrations/`. goose запускает их автоматически при старте приложения.

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

## Лицензия

MIT
