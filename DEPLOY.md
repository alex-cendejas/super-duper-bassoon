# Deployment Guide — super-duper-bassoon PoC

## Prerequisites

- Docker Engine 24+
- Docker Compose v2 (the `docker compose` plugin, **not** the legacy `docker-compose`)

Verify:
```bash
docker compose version
```

## Quick Start

From the project root, run:

```bash
docker compose up --build
```

On the first run this builds all four images (a few minutes). Subsequent runs reuse cached layers and start in seconds.

Once all services are healthy, open the WebUI at:

**http://localhost:3000**

## Services

| Service | Description | Exposed |
|---------|-------------|---------|
| `webui` | Web dashboard (nginx + built Vite SPA) | http://localhost:3000 |
| `server` | Go automation engine + REST API | internal (port 8080) |
| `super-client` | 20 simulated toy clients | – |
| `nats` | NATS 2.x messaging broker | internal (port 4222) |

The REST API is reachable through the nginx proxy at `http://localhost:3000/api/...`.  
Example: `curl http://localhost:3000/api/status`

## Startup Sequence

Docker Compose enforces this dependency chain via health checks:

```
nats (healthy) → server (healthy) → super-client + webui
```

The 20 inner toy clients connect to NATS and start sending results to the server as soon as the server is ready.

## Configuration

All configuration is via environment variables in `docker-compose.yml`. Defaults are tuned for the PoC.

| Variable | Default | Description |
|----------|---------|-------------|
| `POOL_SIZE` | `20` | Number of inner toy clients |
| `CLIENT_PREFIX` | `client` | Prefix for generated client IDs (`client-1` … `client-20`) |
| `LOOP_THRESHOLD_MS` | `5000` | Time window (ms) used to detect client re-entry loops |
| `HEALTH_WINDOW_SIZE` | `10` | Number of past runs used for health aggregation |
| `HEALTH_SUCCESS_THRESHOLD` | `70` | Success % below which circuit breaker consideration starts |
| `CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | `70` | Success % that trips the circuit breaker |
| `CIRCUIT_BREAKER_COOLDOWN_MS` | `60000` | Cooldown before a tripped workflow can be re-evaluated |
| `LOG_LEVEL` | `info` | One of `debug`, `info`, `warn`, `error` |

Edit `docker-compose.yml` and restart to apply changes.

## Data Persistence

The SQLite database is stored in a Docker named volume (`server-data`). Banned clients, workflow definitions, run history, and circuit-breaker state all survive container restarts.

## Useful Commands

```bash
# Start in the background
docker compose up --build -d

# Tail all logs
docker compose logs -f

# Tail a specific service
docker compose logs -f server
docker compose logs -f super-client

# Stop everything (preserves the data volume)
docker compose down

# Stop and wipe all data (fresh start)
docker compose down -v

# Rebuild a single image after a code change
docker compose build server
docker compose up -d --no-deps server
```

## Files Created for This Deployment

```
Dockerfile.server       Go multi-stage build for the automation server
Dockerfile.superclient  Go multi-stage build for the toy client pool
Dockerfile.webui        Node build + nginx image for the web dashboard
nginx.conf              Nginx config: proxies /api/* to server, SPA fallback
docker-compose.yml      Orchestrates all four services with health-check ordering
.dockerignore           Keeps build contexts lean (excludes .git, node_modules, etc.)
DEPLOY.md               This file
```
