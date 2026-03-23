# API de Pedidos

Serviço de gestão de pedidos (e-commerce) em Go, com **Clean Architecture**, **CQRS**, **DDD** e **Outbox Pattern**. Expõe uma API REST documentada com Swagger e um worker que publica eventos no RabbitMQ.

## Tecnologias

- **Go 1.25** — linguagem
- **Gin** — HTTP API
- **MongoDB** — persistência (com replica set para transações)
- **RabbitMQ** — mensageria (eventos de pedido)
- **Swagger (swaggo)** — documentação da API
- **Zap** — logging (JSON com `LOG_FORMAT=json` ou `APP_ENV=production`)
- **Elastic Stack** — ELK (Logstash em prod com GELF), **APM** (tracing), **Metricbeat** (métricas Docker)

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
   | `LOG_FORMAT`       | `json` força logs JSON (útil com ELK); padrão é console colorido em dev | `json` |
   | `ELASTIC_APM_*`    | Opcional fora do Compose; no Docker já vêm no `docker-compose.*.yml` | ver `.env.example` |

3. Com **Docker** (produção ou dev):

   **Produção** (binários + mesma stack de observabilidade que o dev):
   ```bash
   make docker-up
   ```

   **Desenvolvimento** (Air + volume + observabilidade):
   ```bash
   make docker-dev-up
   ```

   A API ficará em `http://localhost:8080`. Swagger: `http://localhost:8080/swagger/index.html`.

   Sobe **Elasticsearch**, **Logstash**, **Kibana**, **APM Server** (`:8200`), **Metricbeat**; **Elastic APM** na `api` e no `worker`. Em **dev** e **prod**, logs da app vão por **GELF** → Logstash → índices `rankmyapp-logs-*` (alinhado com **ECS** no pipeline; ver abaixo).

   **`docker compose ... up api worker`** arranca também **Elasticsearch + Logstash** (a `api`/`worker` em dev enviam logs por **GELF** para `127.0.0.1:12201`, como em prod). **Não** esperam pelo APM Server. Para traces APM no Kibana, sobe o stack completo (`make docker-dev-up`) ou o serviço `apm-server`.

   - **Kibana**: http://localhost:5601 — **APM** (traces), **Discover** (logs `rankmyapp-logs-*`), métricas `metricbeat-*`  
   - **APM (profundidade do trace):** só o **HTTP** vinha do `apmgin`. Spans **MongoDB** (`orders.*`, `outbox_messages.*`) e **RabbitMQ** (`rabbitmq.publish`) são criados em código (`pkg/apmutil` + repositórios + publisher). O agente Go **não** instrumenta `mongo-driver/v2` nem `amqp091-go` automaticamente.  
   - **Elasticsearch**: http://localhost:9200  
   - Se o Elasticsearch falhar (Linux/WSL): `sudo sysctl -w vm.max_map_count=262144`.
   - **Prod** (`make docker-up`): logs da app via **GELF** em `udp://127.0.0.1:12201` (daemon Docker → Logstash; não uses hostname `logstash` no endereço GELF).

   **Logs da app em dev:** a `api` e o `worker` usam **GELF** → Logstash → **`rankmyapp-logs-*`**. No **Kibana → Discover**, data view `rankmyapp-logs-*`, tempo **`@timestamp`**. O Logstash faz parse do JSON do **Zap** e mapeia para **ECS** (padrão de mercado no Elastic): `log.level`, `message`, `http.request.method`, `http.response.status_code`, `url.path`, `source.ip`, `event.duration` (ns), `event.original`, `ecs.version`, `event.dataset`, `service.name` (a partir do tag GELF). Extras Zap em `rankmyapp.*`. Também: **`container.*`**, **`source.geo.*`** (GeoIP), **`event.kind` / `event.category` / `event.type` / `event.outcome`** (SIEM), **`observer.*`**. Reinicia o Logstash após editar `rankmyapp.conf`; índices antigos podem manter formato antigo até à rotação diária.

   `docker compose logs -f api` **não** mostra saída da app (GELF). Recria os containers após mudanças no Air (`docker compose ... up --build` ou `make docker-dev-up`).

   **Kibana: “Name must match…” para `rankmyapp-logs-*`:** esse padrão **só existe depois do primeiro documento** chegar ao Elasticsearch via Logstash. Se só vês `metricbeat-*` e `traces-apm-*`, os **logs da app ainda não estão a ser indexados**. Faz: `curl -s http://localhost:8080/health` (várias vezes), `make docker-dev-check-elk` e `docker compose -f deployments/docker-compose.dev.yml logs logstash --tail 100` (procura `ERROR` / pipeline failed). **Docker Desktop (Windows/Mac):** cria na **raiz do repo** um ficheiro `.env` com `RANKMYAPP_GELF_ADDR=udp://host.docker.internal:12201` (copia de `deployments/.env.example`) e recria `api`/`worker` (`up -d --build api worker`). O `docker-compose.dev.yml` usa essa variável em `gelf-address`. Enquanto não há índices de log, podes usar data views que já existem: **`metricbeat-*`**, **`traces-apm-*`**.

   **Produção / mercado:** ativa **segurança** no Elastic (TLS + utilizadores) em ambientes expostos; o aviso *“Your data is not secure”* no Kibana desaparece ao configurar autenticação. Usa **ILM** e **retention** nos índices `rankmyapp-logs-*` quando tiveres volume real.

   **Testar ingestão de logs no Elasticsearch** (`rankmyapp-logs-*`): com stack **prod** (`make docker-up`) e GELF ativo, ou manualmente; em **dev** o `make test-elk` já não encontra índices da app só com tráfego HTTP.

   ```bash
   make test-elk
   ```

4. **Sem Docker** (só API e worker locais): tenha MongoDB (replica set) e RabbitMQ rodando e use:

   ```bash
   make run-api      # terminal 1
   make run-worker   # terminal 2
   ```

## Comandos principais (Makefile)

Execute `make` na **raiz do projeto**.

| Comando              | Descrição |
|----------------------|-----------|
| `make` ou `make help` | Ajuda (alvos do Makefile, em português) |
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
| `make docker-up`     | Prod: app + **ELK + APM + Metricbeat** |
| `make docker-down`   | Para todos os containers |
| `make docker-reup`   | Rebuild + docker-up (prod) |
| `make docker-dev-up` | Stack **dev** com Air, `--build`; logs da app via GELF → Kibana; ELK + APM + Metricbeat |
| `make docker-dev-down` | Para o stack dev |
| `make docker-dev-check-elk` | Lista índices `rankmyapp*` no Elasticsearch (diagnóstico do data view) |
| `make test-elk`      | Testa ingestão de logs no Elasticsearch (API em execução) |

> **Nota:** **Prod** e **dev** usam a mesma stack de observabilidade declarada nos composes (`deployments/docker-compose.prod.yml` e `deployments/docker-compose.dev.yml`).

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
├── deployments/      # Docker Compose prod/dev + configs ELK (elk/)
├── docs/             # Swagger gerado (`make swag`)
├── scripts/          # test-elk.sh (smoke test de logs)
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
