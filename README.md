# Tuchka Server

Backend service for secure file storage and management written in Go.

## Features

- JWT authentication
- Multipart file uploads
- ZIP archive downloads
- Batch file deletion
- Context-aware request cancellation
- Graceful shutdown
- Structured logging
- Swagger/OpenAPI documentation


## Architecture

```text
HTTP Handlers
   ↓
Service Layer
   ↓
Storage / Repository
```

## Tech Stack

- Go
- Chi Router
- PostgreSQL
- JWT
- slog
- Swagger/OpenAPI
- Docker
- Docker Compose

## Configuration

The project uses a hybrid configuration approach:

- `.env` → secrets and infrastructure configuration
- `yaml` → application configuration

### Example `.env`

```env
CONFIG_PATH="./example/config.yml"
JWT_SECRET="example"
JWT_EXPIRATION_HOURS=24

DB_USER="example"
DB_PASSWORD="example"
DB_NAME="example"
DB_PORT=5432
DB_SSLMODE="disable"
```

### Example `config.yaml`

```yaml
env: local

http_server:
  address: ":8080"
  request_read_timeout: 10s
  response_write_timeout: 30s
  idle_timeout: 60s
  cert_file: "./certs/server.crt"
  key_file: "./certs/server.key"

files:
  storage_dir: "./storage"
  max_download: 30
  max_delete: 30
```

## Run Locally

### 1. Clone repository

```bash
git clone https://github.com/antongolenev23/tuchka-server
cd tuchka-server
```

### 2. Create `.env` and `config/config.yml`

### 3. Run service

```bash
make run
```

## Swagger API Documentation

Swagger UI (local only):

```text
http://localhost:8080/swagger/index.html
```
Also can use docs/swagger.json or docs/swagger.yaml

## Authentication

Protected endpoints require JWT token:

```http
Authorization: Bearer <token>
```

## Testing

Run tests:

```bash
make test
```


## License

MIT
