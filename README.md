# Distributed Cache

A simple distributed in-memory cache built with Go, using consistent hashing for key distribution.

## Features

- In-memory key-value storage with TTL support
- Consistent hashing for even key distribution
- HTTP API for cache operations
- Automatic cleanup of expired keys

## Quick Start

```bash
# Initialize project
go mod init distributed-cache
go mod tidy

# Start 3 cache nodes (in separate terminals)
go run main.go -addr=:8080
go run main.go -addr=:8081
go run main.go -addr=:8082
```

## Usage

### Run Tests

```bash
go run test_client.go
```

### HTTP API

```bash
# Set
curl -X POST http://localhost:8080/set \
  -H "Content-Type: application/json" \
  -d '{"key":"test","value":"hello","ttl":60}'

# Get
curl "http://localhost:8080/get?key=test"

# Delete
curl -X DELETE "http://localhost:8080/delete?key=test"
```

## Architecture

- **Cache**: Thread-safe local key-value store
- **Consistent Hashing**: Distributes keys across nodes with 150 virtual nodes per physical node
- **Cluster**: Manages node membership
- **HTTP Protocol**: Simple REST API for operations

## License

MIT
