# LinkTorch API

A Go backend service for URL analysis and web crawling.

## Getting Started

1. Configure your application in `configs/config.yaml` or via environment variables.
2. Build the application:

   ```bash
   go build -o server cmd/server/main.go
   ```

3. Run with Docker Compose:

   ```bash
   docker-compose up --build
   ```

4. Access the API at `http://localhost:8080/api/v1/users`.

## Project Structure

- `cmd/server` - application entrypoint
- `configs` - configuration files
- `internal` - application code (server, handler, service, repository, model, middleware)
- `pkg/errors` - shared error helpers
- `tests` - integration and e2e tests
