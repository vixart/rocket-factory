# Rocket Factory 🚀

![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/vixart/67f50bd37b7dc5e95c8feae5fb8f7228/raw/coverage.json)

Учебный проект микросервисной архитектуры на Go: «завод по сборке космических кораблей».
Пользователь регистрируется, создаёт заказ из набора деталей, оплачивает его — после чего корабль
асинхронно собирается, а статус заказа обновляется через события Kafka.

Репозиторий построен как **Go workspace** (`go.work`) из независимых модулей: пять сервисов,
общая платформенная библиотека и общие контракты API.

---

## Архитектура

```mermaid
flowchart LR
    Client([Client]) -->|HTTP REST| Nginx[Nginx LB<br/>:8080]
    Nginx --> Order

    subgraph Services
        Order[Order<br/>HTTP, x3 реплики]
        Inventory[Inventory<br/>:50051 gRPC]
        Payment[Payment<br/>:50052 gRPC]
        IAM[IAM<br/>:50053 gRPC]
        Assembly[Assembly<br/>Kafka worker]
    end

    Order -->|gRPC| Inventory
    Order -->|gRPC| Payment
    Order -->|gRPC Whoami| IAM
    Inventory -->|gRPC Whoami| IAM

    Order -->|order.paid| Kafka{{Kafka}}
    Kafka -->|order.paid| Assembly
    Assembly -->|assembly.ship-assembled| Kafka
    Kafka -->|assembly.ship-assembled| Order

    Order --- PGO[(PostgreSQL<br/>:55432)]
    Inventory --- PGI[(PostgreSQL<br/>:55433)]
    IAM --- PGA[(PostgreSQL<br/>:55434)]
    IAM --- Redis[(Redis<br/>:6380)]

    Services -.OTLP.-> OTel[OTel Collector<br/>:4317]
```

### Сервисы

| Сервис        | Протокол        | Порт  | Хранилище                  | Назначение                                                                   |
|---------------|-----------------|-------|----------------------------|------------------------------------------------------------------------------|
| **order**     | HTTP (OpenAPI)  | 8080  | PostgreSQL `:55432`        | Создание, получение, оплата и отмена заказов; оркестрация остальных сервисов  |
| **inventory** | gRPC            | 50051 | PostgreSQL `:55433`        | Каталог деталей, проверка совместимости, резервирование/коммит/освобождение   |
| **payment**   | gRPC            | 50052 | —                          | Проведение оплаты заказа                                                     |
| **iam**       | gRPC            | 50053 | PostgreSQL `:55434`, Redis | Регистрация, логин, сессии, `Whoami` для остальных сервисов                   |
| **assembly**  | Kafka consumer  | —     | —                          | Асинхронная «сборка корабля» по событию оплаты                                |

В контейнерном режиме (`task up-all`) Order поднимается в трёх репликах за Nginx на `:8080`,
а входящий HTTP-трафик ограничивается распределённым rate limiter'ом на Redis
(`redis-ratelimit:6379`, по умолчанию 100 rps при burst 200).

Вспомогательные модули:

- **platform** — переиспользуемая инфраструктура: логгер, трейсинг, метрики, Kafka producer/consumer,
  Redis-клиент, graceful shutdown (`closer`), gRPC health, auth-контекст, middleware.
- **shared** — контракты: `.proto` (gRPC), OpenAPI-спека Order API и сгенерированный по ним Go-код.

### Бизнес-поток

1. `POST /api/v1/orders` — Order проверяет сессию в IAM, валидирует совместимость деталей
   и резервирует их в Inventory. Заказ переходит в `PENDING_PAYMENT`.
2. `POST /api/v1/orders/{uuid}/pay` — в одной транзакции (с блокировкой заказа `SELECT ... FOR UPDATE`)
   Order вызывает Payment, переводит заказ в `PAID` и публикует событие `order.paid` в Kafka.
3. **Assembly** читает `order.paid`, «собирает корабль» и публикует `assembly.ship-assembled`.
4. Order-консьюмер читает `assembly.ship-assembled`, коммитит резерв деталей в Inventory
   и переводит заказ в `ASSEMBLED` (идемпотентно — повторные сообщения игнорируются).
5. `POST /api/v1/orders/{uuid}/cancel` — отмена заказа с освобождением резерва (`CANCELLED`).

Аутентификация сквозная: клиент передаёт токен сессии, Order-middleware и gRPC-интерсепторы
пробрасывают его дальше по цепочке вызовов, каждый сервис валидирует сессию через IAM.

---

## Стек

- **Go 1.26**, multi-module workspace
- **gRPC** + Protocol Buffers (`buf`, `protoc-gen-go`, `protoc-gen-go-grpc`)
- **OpenAPI 3** → Go-код через [`ogen`](https://github.com/ogen-go/ogen)
- **PostgreSQL** + миграции [`goose`](https://github.com/pressly/goose)
- **Redis** — кэш и хранилище сессий IAM, а также распределённый rate limiter Order API
- **Nginx** — балансировка HTTP-трафика между репликами OrderService
- **Kafka** (KRaft, без ZooKeeper) — асинхронные события
- **OpenTelemetry** → OTel Collector → Jaeger (трейсы), Prometheus + Grafana (метрики),
  Elasticsearch + Kibana (логи)
- **Docker Compose** для локальной инфраструктуры, **Task** ([go-task](https://taskfile.dev)) вместо Makefile
- **mockery**, **testify**, **testcontainers** — тестирование; **vegeta** — нагрузочные тесты
- **golangci-lint**, **gofumpt**, **gci** — качество кода

---

## Структура репозитория

```
.
├── assembly/          # сервис сборки (Kafka consumer + producer)
├── iam/               # сервис аутентификации и пользователей
├── inventory/         # сервис каталога деталей
├── order/             # сервис заказов (HTTP API, оркестратор)
├── payment/           # сервис оплаты
├── platform/pkg/      # общая инфраструктура (logger, tracing, metrics, kafka, redis, closer...)
├── shared/
│   ├── proto/         # .proto-контракты (auth, user, inventory, payment, events, common)
│   ├── api/           # OpenAPI-спека Order API
│   └── pkg/           # сгенерированный Go-код
├── migrations/        # SQL-миграции по сервисам (goose)
├── deploy/
│   ├── compose/       # docker-compose: core, observability, order, inventory, iam, services
│   ├── dockerfiles/   # образы сервисов + migrator.Dockerfile (goose)
│   ├── nginx/         # конфигурация load balancer'а перед OrderService
│   ├── grafana/       # provisioning и дашборды
│   └── otel/          # конфигурация OTel Collector
├── .github/workflows/ # CI: build, lint, unit / api / e2e тесты
└── Taskfile.yaml      # все команды проекта
```

Каждый сервис следует единой структуре:

```
<service>/
├── cmd/               # точка входа
├── internal/
│   ├── app/           # сборка приложения и DI-контейнер
│   ├── api/           # транспортный слой (gRPC / HTTP handlers) + конвертеры
│   ├── service/       # бизнес-логика
│   ├── repository/    # доступ к данным + конвертеры record ↔ model
│   ├── client/        # gRPC-клиенты к другим сервисам
│   ├── model/         # доменные модели
│   ├── config/        # конфигурация (env + yaml)
│   └── errors/        # доменные ошибки
└── config.{local,staging,production}.yaml
```

---

## Быстрый старт

Требования: Go 1.26+, Docker, [Task](https://taskfile.dev/installation/).

### Вариант 1 — всё в контейнерах

```bash
task setup     # инструменты разработки (buf, ogen, golangci-lint, gofumpt, gci)
task up-all    # core + observability + БД с миграциями + сервисы за Nginx (order ×3)
task kibana:init   # Data View для просмотра логов (один раз)
```

`up-all` поднимает БД до healthy и прогоняет одноразовые `migrator-*` контейнеры (goose),
так что миграции накатываются сами. Index template для логов создаётся автоматически внутри `up-core`.

### Вариант 2 — инфраструктура в контейнерах, сервисы локально

```bash
task setup
task up-core        # сеть, Kafka + Kafka UI, observability, Redis для rate limiter
task up-inventory   # PostgreSQL + миграции
task up-order
task up-iam         # PostgreSQL + Redis + миграции

# сервисы (каждый в своём терминале)
task run:inventory   # :50051
task run:payment     # :50052
task run:iam         # :50053
task run:order       # :8080
task run:assembly    # Kafka worker

task kibana:init
```

Остановить всё: `task down-all`.

### Точки доступа

| Что                  | URL                                              |
|----------------------|--------------------------------------------------|
| Order API (за Nginx) | http://localhost:8080/api/v1/orders              |
| Kafka UI             | http://localhost:8090                            |
| Jaeger (трейсы)      | http://localhost:16686                           |
| Grafana (метрики)    | http://localhost:3000 — `admin` / `admin`        |
| Prometheus           | http://localhost:9090                            |
| Kibana (логи)        | http://localhost:5601                            |
| Elasticsearch        | http://localhost:9200                            |

---

## Команды разработки

Полный список — `task --list`.

### Кодогенерация

```bash
task gen              # весь сгенерированный код (proto + openapi)
task proto:gen        # Go-код из .proto
task ogen:gen         # Go-код из OpenAPI
task mocks:gen        # моки интерфейсов (mockery)
```

### Качество кода

```bash
task format           # gofumpt + gci
task lint             # golangci-lint
task build            # go build всех модулей
```

### Тесты

```bash
task test             # unit-тесты с race-детектором
task test:coverage    # покрытие бизнес-логики (порог — 40%)
task coverage:html    # HTML-отчёт покрытия
task test:api         # API-тесты
task test:e2e         # e2e order с реальной Kafka (Redpanda) через testcontainers
task load:http        # нагрузочный тест vegeta: 50 → 500 RPS, результаты — в Grafana
```

### Миграции

В `up-*` миграции накатывает контейнер `migrator-<сервис>` (goose внутри Docker).
Эти задачи — для ручной работы с goose с хоста:

```bash
task migrate:all:up                      # накатить все миграции всех сервисов
task migrate:order:up                    # / :down / :status
task migrate:order:create -- <имя>       # новый файл миграции
```

Аналогично для `inventory` и `iam`.

### Инфраструктура

```bash
task up-core / down-core            # общая сеть, Kafka, observability, Redis rate limiter
task up-order / down-order          # PostgreSQL + миграции Order
task up-inventory / down-inventory
task up-iam / down-iam              # PostgreSQL + Redis + миграции
task up-all / down-all              # всё сразу + контейнеризованные сервисы за Nginx
```

---

## Конфигурация

Конфигурация читается из YAML-файла (`config.local.yaml`, путь задаётся `CONFIG_PATH`)
и переменных окружения из `*.env`:

- `core.env` (в корне) — порты и версии образов общей инфраструктуры (Kafka, observability,
  Nginx, Redis), а также build args `GO_IMAGE` / `ALPINE_IMAGE` / `GOOSE_VERSION`. Подключён
  в `Taskfile.yaml` через `dotenv`, поэтому версии живут в одном месте — и для Docker-сборок,
  и для хост-инсталла goose;
- `<service>/<service>.env` — параметры БД, порты сервиса, адреса зависимостей, версии образов.
  Лежит рядом с сервисом и является единственным источником: его читает сам сервис при запуске
  из IDE (`godotenv`), `docker compose` при подъёме зависимостей и задачи `migrate:*`
  (DSN для goose собирается из `POSTGRES_*`).

Профили окружений: `config.local.yaml` (запуск с хоста), `config.docker.yaml` (запуск в контейнере),
`config.staging.yaml`, `config.production.yaml`.

---

## CI

GitHub Actions (`.github/workflows/ci.yml`) на каждый PR запускает: `build`, `lint`,
unit-тесты с покрытием, API-тесты и e2e-тесты. Покрытие публикуется бейджем в шапке README.

---

## Лицензия

[MIT](LICENSE)
