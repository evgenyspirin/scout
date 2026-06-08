# Scout — Greenhouse Pest & Disease Monitoring

Scout is the early-warning system for the **AlfaGreen** greenhouse.   
Cameras photograph the crop, an ML model flags pests and   
diseases with bounding boxes, and growers open Scout to:

- browse a **gallery** of photos with detection boxes drawn on top, filter by
  class/confidence, and open any photo full-size to inspect its predictions;
- view a **greenhouse map** showing where the problems are, with zoom/pan, where
  clicking a point filters the gallery to nearby photos.

---
### Short architecture overview

- **Backend** — Go + [Fiber v2], DDD / Clean Architecture, JWT auth, structured
  logging (zap), Prometheus metrics, graceful shutdown. Reads photos/predictions
  from SQLite, serves originals from MinIO, and generates thumbnails on demand.
- **Thumbnail engine**: lazy generation, two-level cache (in-memory byte-budgeted
  LRU + Redis with `allkeys-lru`), `singleflight` de-duplication of identical
  concurrent requests, and a semaphore bounding concurrent decode/resize to avoid
  memory spikes on ~1 vCPU / 1 GB RAM. Aspect ratio preserved, never upscaled.
- **Frontend** — React 19 + TypeScript + Vite + Redux Toolkit + react-konva, with
  CSS Modules and vitest.
- **Storage** — MinIO (object storage) and Redis (thumbnail cache).

## Backend

**The application is built on "The Twelve-Factor App" rules:**

- **One code base** – GitHub
- **Clearly declared and isolated dependencies** – `go.mod`
- **Configuration** must be located in environment variables – Helm(values)->K8S(configmap/secret)->expvars->config.yml{val})
- **Strict separation of build, release, and execution** – CI/CD pipeline (future)
- **Stateless processes** – we store data in constant storage and update it (**DB**)
- **Port binding** – the built-in web server runs on the specific port from the environment variable
- **Concurrency** – processes can be split into separate microservices in the future in case of high-volume traffic(scaling)
- **Disposability** – supports graceful shutdown through a single `Context`
- **Logs** – currently using **Zap Logger**, future: **ELK Stack** (Elasticsearch + Logstash + Kibana)
- **Admin processes** – QA/Dev/Prod environments must be as similar as possible (future)

Interfaces: “Go interfaces generally belong in the package that uses values of the interface type, not the package that implements those values.”
- https://go.dev/wiki/CodeReviewComments


**The application uses "DDD (Domain Driven Design)" / Clean Architecture with fully separated layers:**

- **Interface** – REST, controllers, middlewares, HTTP request/response DTOs
- **Application** – use cases, calls domain objects and repositories through interfaces
- **Domain** – domain entities (currently no complex business rules)
- **Infrastructure** – repository implementations, DB models, storages (SQLite, Redis,MinIO), clients (HTTP...)

**Request flow:**  
```
client → interface (REST/Fiber) → application (use cases) → domain → infrastructure (SQLite / MinIO / Redis)
```

**Dependencies** are directed inward: outer layers depend on inner ones, not vice versa.

---

### Backend structure

The application has a structure based on Go convention:
- https://github.com/golang-standards/project-layout

```
scout/
├── backend/                 Go service + seeder
│   ├── cmd/scout             backend entrypoint
│   ├── cmd/seeder-images     idempotent image seeder
│   ├── config                env-based configuration
│   └── internal/
│       ├── domain            entities (photo, user)
│       ├── application       use cases (photoapp, thumbapp, authapp) + interfaces
│       ├── infrastructure    sqlite, minio, redis/lru cache, thumbnail gen, jwt, metrics
│       └── interface/api/rest controllers, middleware, dto (easyjson), validator, api-specs
├── frontend/                React + TS app (gallery, map, filters, login)
├── dataset/                 predictions.db + images/ (provided)
├── docker-compose.yml
└── Makefile
```
---

## API

Full contract: [`backend/internal/interface/api/rest/api-specs/openapi/scout/openapi.yaml`](scout/backend/internal/interface/api/rest/api-specs/openapi/scout/openapi.yaml).   
Have fun with ready-to-run example requests:[`backend/internal/interface/api/rest/api-specs/scout_api.http`](backend/internal/interface/api/rest/api-specs/scout_api.http).

| Method | Path | Access |
|---|---|---|
| POST | `/api/v1/auth/login` | public |
| GET  | `/api/v1/photos` | user |
| GET  | `/api/v1/photos/{id}` | user |
| GET  | `/api/v1/photos/{id}/thumbnail` | user (token via header or `?token=`) |
| GET  | `/api/v1/photos/{id}/original` | user (token via header or `?token=`) |
| POST | `/api/v1/photos/{id}/upload-link` | admin |
| HEAD | `/api/v1/photos/{id}/object` | admin |
| GET  | `/healthz` | public |
| GET  | `/metrics` | admin |
| GET  | `/debug/vars` | admin |

---

## Required tools

- **Docker + Docker Compose** (the simplest path — runs everything).
- For running pieces locally instead: **Go 1.25+**, **Node 20+** with **Yarn**.
- For linting locally: **golangci-lint** and the frontend ESLint (installed via yarn).

---

## Quick start

```bash
$ cd scout
$ make up                      # builds & starts backend, frontend, MinIO, Redis, then seeds images
$ make down                    # stop the app
```

For more details:
```bash
$ make help
```

Then open:

- **Frontend** — http://localhost:5173
- **Backend**  — http://localhost:8080/healthz
- **MinIO console** — http://localhost:9001 (`minioadmin` / `minioadmin`)

### Default credentials

| Login    | Password    | Role  |
|----------|-------------|-------|
| `insect` | `insect123` | user  |
| `admin`  | `admin123`  | admin |

`admin` can additionally create upload links and read `/metrics` & `/debug/vars`.


