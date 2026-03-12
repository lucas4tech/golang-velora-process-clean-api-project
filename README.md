# RankMyApp — API de Pedidos

Serviço de gestão de pedidos (e-commerce) em Go, com **Clean Architecture**, **CQRS**, **DDD** e **Outbox Pattern**. Expõe uma API REST documentada com Swagger e um worker que publica eventos no RabbitMQ.

## Tecnologias

- **Go 1.25** — linguagem
- **Gin** — HTTP API
- **MongoDB** — persistência (com replica set para transações)
- **RabbitMQ** — mensageria (eventos de pedido)
- **Swagger (swaggo)** — documentação da API
- **Zap** — logging

## Arquitetura

- **API** (`cmd/api`): REST em `:8080`, CRUD de pedidos e health check.
- **Worker** (`cmd/worker`): lê mensagens da tabela outbox e publica no RabbitMQ em intervalo configurável.
- **Domain**: entidades e value objects de pedido; eventos de domínio (criado, status alterado).
- **Application**: use cases (commands e queries) e DTOs.
- **Infra**: handlers HTTP, repositórios MongoDB, publisher RabbitMQ, unit of work, outbox worker.

## Tomadas de decisão arquiteturais

As decisões abaixo explicam **o quê** foi escolhido e **por quê**, para facilitar manutenção e onboarding.

| Decisão | Escolha | Motivo |
|--------|---------|--------|
| **Estilo arquitetural** | Clean Architecture (camadas domain → app → infra) | Manter domínio isolado de frameworks e DB; testes unitários sem infra; troca de persistência/mensageria sem alterar regras. |
| **Separação leitura/escrita** | CQRS (Commands vs Queries) | Escrita (criar pedido, atualizar status) e leitura (listar, buscar por ID) com handlers distintos; evolução independente e clareza de intenção. |
| **Modelagem do domínio** | DDD (entidades, value objects, eventos de domínio) | Pedido como agregado; status e itens como value objects; eventos `OrderCreated` e `OrderStatusChanged` para integração e auditoria. |
| **Consistência + mensageria** | Outbox Pattern | Gravar pedido e mensagens na mesma transação (MongoDB); worker lê a tabela outbox e publica no RabbitMQ. Evita “pedido salvo mas evento perdido” sem 2PC. |
| **Transações** | MongoDB com replica set | Driver exige replica set para transações multi-documento; no Docker o compose sobe um nó com `rs0` e `init-mongodb` executa `rs.initiate()`. |
| **Mensageria** | RabbitMQ (exchange + publish) | Desacoplamento entre API e consumidores; eventos de pedido reutilizáveis (notificações, estoque, analytics). |
| **Processamento assíncrono** | Worker separado do processo da API | Publicação no RabbitMQ fora do request HTTP; falhas de publish não derrubam a API; reprocessamento e backoff no worker. |
| **API HTTP** | Gin + handlers finos | Handlers só validam entrada e chamam use cases; regras e erros de negócio ficam na camada de aplicação. |
| **Contratos da API** | DTOs (request/response) + Swagger | Contratos estáveis e documentados; geração de docs a partir do código (`make swag`). |
| **Erros na API** | Tipos de erro (pkg/errors) com código e status HTTP | Respostas 4xx/5xx previsíveis; middleware de erro centraliza formatação e logging. |
| **Testes unitários** | Apenas `./internal/...` | Foco em regras e use cases; `cmd/`, adaptadores de infra pesados e mocks ficam de fora do coverage unitário. |
| **Testes de integração** | Testcontainers (RabbitMQ) | Cenários com broker real sem depender de ambiente; CI roda com Docker disponível no runner. |

## Pré-requisitos

- Go 1.25+
- MongoDB (com replica set se for usar transações; no Docker o compose já configura)
- RabbitMQ (opcional para apenas rodar a API; necessário para o worker e para publicar eventos)

## Configuração

1. Clone o repositório e entre na pasta do projeto.

2. Copie o arquivo de ambiente de exemplo e ajuste se precisar:

   ```bash
   cp .env.example .env
   ```

   Variáveis (ver `.env.example`):

   | Variável         | Descrição                          | Exemplo                          |
   |------------------|------------------------------------|----------------------------------|
   | `PORT`           | Porta da API                       | `8080`                           |
   | `MONGO_URI`      | URI do MongoDB (replica set: `?replicaSet=rs0`) | `mongodb://localhost:27017/?replicaSet=rs0` |
   | `MONGO_DATABASE` | Nome do banco                      | `rankmyapp`                      |
   | `RABBITMQ_URL`   | URL do RabbitMQ                    | `amqp://guest:guest@localhost:5672/` |
   | `RABBITMQ_EXCHANGE` | Exchange de eventos            | `orders.events`                  |

3. Com **Docker** (produção ou dev):

   **Produção** (binários, sem hot reload):
   ```bash
   make docker-up
   ```

   **Desenvolvimento** (Air + volume, hot reload):
   ```bash
   make docker-dev-up
   ```

   A API ficará em `http://localhost:8080`. Swagger: `http://localhost:8080/swagger/index.html`.

4. **Sem Docker** (só API e worker locais): tenha MongoDB (replica set) e RabbitMQ rodando e use:

   ```bash
   make run-api      # terminal 1
   make run-worker   # terminal 2
   ```

## Comandos principais (Makefile)

Execute `make` na **raiz do projeto**.

| Comando              | Descrição |
|----------------------|-----------|
| `make help`          | Lista todos os alvos |
| `make build`         | Gera binários em `./bin/` (api e worker) |
| `make run-api`       | Sobe a API (`go run ./cmd/api`) |
| `make run-worker`    | Sobe o worker (`go run ./cmd/worker`) |
| `make dev`           | API + Worker com hot reload (Air) |
| `make dev-api`       | Só API com hot reload |
| `make dev-worker`    | Só worker com hot reload |
| `make test-unit`     | Testes unitários (`./internal/...`) |
| `make test-unit-cover` | Testes unitários com cobertura → `coverage.out` |
| `make coverage`      | Testes com cobertura + exibe total % |
| `make test-integration` | Testes de integração (testcontainers, RabbitMQ) |
| `make test`          | Unit + integration |
| `make lint`          | golangci-lint |
| `make swag`          | Regenera documentação Swagger em `docs/` |
| `make tidy`          | `go mod tidy` |
| `make clean`         | Remove `bin/` e arquivos de cobertura |
| `make docker-build`  | Build das imagens de **produção** (API e Worker) |
| `make docker-up`     | Sobe os containers de **produção** |
| `make docker-down`   | Para os containers |
| `make docker-reup`   | Rebuild + docker-up (prod) |
| `make docker-dev-build` | Build das imagens de **dev** (Go + Air) |
| `make docker-dev-up` | Sobe o stack de **dev** com hot reload (Air) |
| `make docker-dev-down` | Para o stack de dev |

> **Nota:** **Prod:** `docker-compose.prod.yml` + `Dockerfile.prod` (binários). **Dev:** `docker-compose.dev.yml` + `Dockerfile.dev` (Air + volume). Execute `make docker-up` ou `make docker-dev-up` na raiz do projeto.

## API — Endpoints

| Método | Endpoint                    | Descrição |
|--------|-----------------------------|-----------|
| GET    | `/health`                   | Health check |
| GET    | `/swagger/index.html`      | Documentação Swagger |
| POST   | `/api/v1/orders`           | Cria pedido |
| GET    | `/api/v1/orders`           | Lista pedidos (query: `customer_id`, `status`, `limit`, `offset`) |
| GET    | `/api/v1/orders/:id`       | Busca pedido por ID |
| PATCH  | `/api/v1/orders/:id/status` | Atualiza status (ex.: `created`, `processing`, `shipped`, `delivered`, `cancelled`) |


## Estrutura do projeto

```
.
├── cmd/
│   ├── api/          # Entrada da API HTTP
│   └── worker/       # Entrada do worker outbox
├── configs/          # Carregamento de configuração
├── deployments/      # Docker: prod (docker-compose.prod.yml, Dockerfile.prod) e dev (docker-compose.dev.yml, Dockerfile.dev)
├── docs/             # Swagger gerado (make swag)
├── internal/
│   ├── app/          # Commands, queries, DTOs, use cases
│   ├── domain/       # Entidades, eventos, repositórios (interfaces), value objects
│   └── infra/        # HTTP, MongoDB, RabbitMQ, unit of work, outbox worker
├── pkg/              # Logger, errors (uso transversal)
├── test/
│   └── integration/ # Testes de integração (testcontainers)
├── .env.example
├── Makefile
└── README.md
```

## Testes

- **Unitários**: `make test-unit` ou `make test-unit-cover` (apenas `./internal/...`).
- **Cobertura**: `make coverage` mostra o percentual total; `go tool cover -func=coverage.out` mostra por arquivo.
- **Integração**: `make test-integration` (requer Docker para testcontainers com RabbitMQ).

## CI/CD

O workflow em `.github/workflows/ci.yml` executa:

1. **Lint** — golangci-lint
2. **Unit tests** — testes com cobertura e upload do artefato `coverage.out`
3. **Integration tests** — testes com tag `integration`
