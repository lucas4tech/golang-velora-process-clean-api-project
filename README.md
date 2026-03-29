# API de Pedidos

Serviço de gestão de pedidos (e-commerce) em Go, com **Clean Architecture**, **CQRS**, **DDD** e **Outbox Pattern**. API REST com Swagger e worker que publica eventos no RabbitMQ.

## Conteúdo

- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Tomadas de decisão arquiteturais](#tomadas-de-decisão-arquiteturais)
- [Pré-requisitos](#pré-requisitos)
- [Configuração](#configuração)
- [Observabilidade (logs, APM, métricas)](#observabilidade-logs-apm-métricas)
- [Comandos principais (Makefile)](#comandos-principais-makefile)
- [API — Endpoints](#api--endpoints)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Testes](#testes)
- [CI/CD](#cicd)

## Tecnologias

| Área | Stack |
|------|--------|
| Linguagem | **Go 1.25** |
| HTTP | **Gin** |
| Dados | **MongoDB** (replica set para transações) |
| Mensageria | **RabbitMQ** |
| Documentação | **Swagger (swaggo)** |
| Logs | **Zap** (JSON com `LOG_FORMAT=json` ou `APP_ENV=production`) |
| Observabilidade | **Elasticsearch**, **Logstash**, **Kibana**, **APM Server**, **Metricbeat** |

## Arquitetura

- **API** (`cmd/api`): REST em `:8080`, pedidos e health check.
- **Worker** (`cmd/worker`): lê a outbox no MongoDB e publica no RabbitMQ.
- **Domain**: entidades, value objects, eventos de domínio.
- **Application**: use cases (commands/queries) e DTOs.
- **Infra**: handlers Gin, repositórios MongoDB, publisher RabbitMQ, unit of work, outbox worker, middleware de logs/erros.
- **`pkg/apmutil`**: spans Elastic APM para MongoDB e RabbitMQ (instrumentação manual; o agente Go não cobre `mongo-driver/v2` nem `amqp091-go`).

## Tomadas de decisão arquiteturais

| Decisão | Escolha | Motivo |
|--------|---------|--------|
| **Estilo arquitetural** | Clean Architecture (domain → app → infra) | Domínio isolado de frameworks e DB; testes sem infra pesada. |
| **Leitura/escrita** | CQRS (commands vs queries) | Intenção clara e evolução independente. |
| **Domínio** | DDD (entidades, VOs, eventos) | Pedido como agregado; eventos para integração. |
| **Mensageria fiável** | Outbox Pattern | Pedido + outbox na mesma transação MongoDB; worker publica no RabbitMQ. |
| **Transações** | MongoDB replica set | Obrigatório para transações multi-documento; Compose com `rs0` + `init-mongodb`. |
| **API** | Gin + handlers finos | Validação na borda; regras nos use cases. |
| **Erros** | `pkg/errors` + middleware | 4xx/5xx previsíveis e logging centralizado. |
| **Logs em Docker** | GELF → Logstash → `rankmyapp-logs-*` | Índices diários, pipeline **ECS** (campos normalizados para Kibana/SIEM). |
| **Tracing** | Elastic APM + `apmgin` + spans em repos/RabbitMQ | HTTP automático; DB e fila via `pkg/apmutil`. |
| **Testes unitários** | `./internal/...` | Foco em regras e use cases. |
| **Integração** | Testcontainers (RabbitMQ) | Broker real sem ambiente manual. |

## Pré-requisitos

- Go 1.25+
- MongoDB (replica set para transações; o Compose de desenvolvimento já configura)
- RabbitMQ (necessário para o worker e eventos de pedido)
- Docker (para stack completa e testes de integração)

## Configuração

1. Clona o repositório.

2. Variáveis de ambiente da **aplicação** (local, sem Compose):

   ```bash
   cp .env.example .env
   ```

   | Variável | Descrição | Exemplo |
   |----------|-----------|---------|
   | `PORT` | Porta da API | `8080` |
   | `MONGO_URI` | URI com replica set | `mongodb://localhost:27017/?replicaSet=rs0` |
   | `MONGO_DATABASE` | Nome da base | `rankmyapp` |
   | `RABBITMQ_URL` | AMQP | `amqp://guest:guest@localhost:5672/` |
   | `RABBITMQ_EXCHANGE` | Exchange de eventos | `orders.events` |
   | `LOG_FORMAT` | `json` força logs JSON (ELK/GELF) | `json` |
   | `ELASTIC_APM_*` | APM fora do Compose | ver `.env.example` |

3. **Docker** — ficheiros em `deployments/`:

   | Ambiente | Comando | Compose | Variáveis da app (`env_file` na raiz) |
   |----------|---------|---------|----------------------------------------|
   | Produção | `make docker-up` | `docker-compose.prod.yml` | `.env.production` |
   | Desenvolvimento (Air + ELK) | `make docker-dev-up` | `docker-compose.dev.yml` | `.env.development` |
   | Teste (sem ELK, portas alternativas) | `make docker-test-up` | `docker-compose.test.yml` | `.env.test` |

   Os serviços **`api`** e **`worker`** carregam o ficheiro na **raiz do repositório**; só **`ELASTIC_APM_SERVICE_NAME`** fica no Compose (diferente por serviço). Modelos: **`.env.*.example`**.

   - API dev/prod: **http://localhost:8080** — API stack teste: **http://localhost:8081**
   - **`docker compose ... up api worker`** (dev) arranca também Elasticsearch e Logstash (GELF). Não bloqueia no APM Server.

4. **Sem Docker** (MongoDB + RabbitMQ locais):

   ```bash
   make run-api      # terminal 1
   make run-worker   # terminal 2
   ```

---

## Observabilidade (logs, APM, métricas)

### URLs rápidas

| Serviço | URL |
|---------|-----|
| Kibana | http://localhost:5601 |
| Elasticsearch | http://localhost:9200 |
| APM Server | http://localhost:8200 |

Se o Elasticsearch falhar no Linux/WSL: `sudo sysctl -w vm.max_map_count=262144`.

### Logs da aplicação (Kibana Discover)

- Fluxo: **containers `api` / `worker`** → driver **GELF** (UDP) → **Logstash** (`deployments/elk/logstash/pipeline/rankmyapp.conf`) → índices **`rankmyapp-logs-YYYY.MM.dd`**.
- **Data view** no Kibana: padrão **`rankmyapp-logs-*`**, campo de tempo **`@timestamp`**. O índice só aparece **depois do primeiro evento** indexado.
- O pipeline faz parse do JSON **Zap** e mapeia para **ECS**: `log.*`, `http.*`, `url.*`, `source.*`, `event.duration` (nanosegundos; no Zap `latency` vem em **segundos** decimais), `event.original`, `container.*`, `source.geo.*` (GeoIP em IPs públicos), `event.kind` / `event.category` / `event.type` / `event.outcome`, `observer.*`, `service.name`, `ecs.version`. Campos extra do Zap em `rankmyapp.*`.
- **`docker compose logs -f api` não mostra** a saída da app com GELF — usa o Kibana (ou json-file só se removeres o GELF).
- Após editar `rankmyapp.conf`: `docker compose -f deployments/docker-compose.dev.yml restart logstash`.

**Docker Desktop (Windows / Mac):** se não surgirem índices `rankmyapp-logs-*`, o daemon muitas vezes não entrega GELF em `127.0.0.1:12201`. Na **raiz do repo**, cria `.env` com:

```bash
RANKMYAPP_GELF_ADDR=udp://host.docker.internal:12201
```

(Referência: `.env.example` na raiz.) O `Makefile` passa `--env-file .env` quando esse ficheiro existe, para o Compose interpolar `RANKMYAPP_GELF_ADDR`. Sem `make`, o Compose usa por defeito `deployments/.env` para essas variáveis. Recria `api` e `worker`. Em Linux nativo o default `udp://127.0.0.1:12201` costuma bastar.

**Diagnóstico:** `make docker-dev-check-elk` lista índices `rankmyapp*`; `docker compose -f deployments/docker-compose.dev.yml logs logstash --tail 100` para erros de pipeline.

**Produção:** GELF em `udp://127.0.0.1:12201` (daemon Docker → porta publicada do Logstash; não uses o hostname `logstash` no endereço GELF). Para ambientes expostos: TLS + autenticação no Elastic; **ILM** / retenção em `rankmyapp-logs-*`.

### APM / tracing (Kibana Observability → APM)

- **`apmgin`** cria a transação **HTTP** por pedido.
- Spans **MongoDB** (`orders.*`, `outbox_messages.*`) e **RabbitMQ** (`rabbitmq.publish`) estão em **`pkg/apmutil`** e nos repositórios / publisher.
- Fluxo típico no Kibana: **Observability → APM →** serviço **`rankmyapp-api`** ou **`rankmyapp-worker`** → **Transactions** / **Traces**. Gera tráfego e ajusta o intervalo de tempo no canto superior direito.

### Métricas

- **Metricbeat** (módulo Docker, etc.) → índices `metricbeat-*` — data view correspondente no Discover.

### Smoke test de logs no ES

Com a stack **dev** a correr e tráfego na API (ex.: `curl http://localhost:8080/health`), confirma índices com `make docker-dev-check-elk` ou no Kibana (data view `rankmyapp-logs-*`).

---

## Comandos principais (Makefile)

Executa `make` na **raiz do projeto**. Compose: `-f deployments/docker-compose.{prod,dev}.yml`.

| Comando | Descrição |
|---------|-----------|
| `make` / `make help` | Ajuda |
| `make build` | Binários em `./bin/` (api + worker) |
| `make run-api` / `make run-worker` | Executa API ou worker |
| `make dev` / `make dev-api` / `make dev-worker` | Air (hot reload) |
| `make test-unit` / `make test-unit-cover` / `make coverage` | Testes unitários (`./internal/...`) |
| `make test-integration` / `make test` | Integração / tudo |
| `make lint` | golangci-lint |
| `make swag` | Regenera `docs/` (Swagger) |
| `make tidy` / `make clean` | `go mod tidy` / artefactos locais |
| `make docker-build` / `make docker-up` / `make docker-down` / `make docker-reup` | **Produção** |
| `make docker-dev-up` / `make docker-dev-down` | **Dev** (Air + ELK + APM + Metricbeat) |
| `make docker-test-up` / `make docker-test-down` | **Test** (Mongo + RabbitMQ + app prod; `.env.test` na raiz; API `:8081`) |
| `make docker-dev-check-elk` | Lista índices `rankmyapp*` no Elasticsearch |
| `make docker-dev-logs-api` / `make docker-dev-logs-worker` | `compose logs -f` (com GELF, a app pode não aparecer aqui) |

---

## API — Endpoints

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/health` | Health check |
| GET | `/swagger/index.html` | Swagger UI |
| POST | `/api/v1/orders` | Cria pedido |
| GET | `/api/v1/orders` | Lista (`customer_id`, `status`, `limit`, `offset`) |
| GET | `/api/v1/orders/:id` | Pedido por ID |
| PATCH | `/api/v1/orders/:id/status` | Atualiza status |

---

## Estrutura do projeto

```
.
├── cmd/api/                 # API HTTP
├── cmd/worker/              # Worker outbox
├── configs/
├── deployments/             # compose prod/dev/test, Dockerfiles, elk/
├── docs/                    # Swagger (`make swag`)
├── internal/
│   ├── app/                 # Commands, queries, DTOs, use cases
│   ├── domain/
│   └── infra/               # HTTP, MongoDB, RabbitMQ, UoW, outbox worker
├── pkg/
│   ├── apmutil/             # Spans APM (MongoDB, RabbitMQ)
│   ├── errors/
│   └── logger/
├── test/integration/
├── .air.toml / .air.worker.toml
├── .env.example
├── Makefile
└── README.md
```

---

## Testes

- **Unitários:** `make test-unit` ou `make test-unit-cover` (`./internal/...`).
- **Cobertura:** `make coverage` ou `go tool cover -func=coverage.out`.
- **Integração:** `make test-integration` (Docker / testcontainers + RabbitMQ).

---

## CI/CD

O workflow `.github/workflows/ci.yml` executa:

1. **Lint** — golangci-lint  
2. **Unit tests** — cobertura e artefato `coverage.out`  
3. **Integration tests** — tag `integration`
