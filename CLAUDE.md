# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`parsec` is a DHT lookup performance measurement tool that measures `PUT` and `GET` performance of the IPFS Amino DHT and other libp2p-kad-dht networks. The system consists of two main components:

- **Server**: A libp2p peer that participates in the DHT and exposes an HTTP API for publication/retrieval operations
- **Scheduler**: Orchestrates measurement experiments by instructing servers to publish content and retrieve it

The system supports multiple "fleets" (groups of servers with different configurations) for comparative analysis.

## Development Commands

### Building and Testing
```bash
# Build the binary
just build

# Run tests
just test

# Format code
just format

# Clean build artifacts
just clean
```

### Database Operations
```bash
# Start local PostgreSQL database
just database

# Install required tools (migrate, sqlboiler)
just tools

# Reset database (down + up + regenerate models)
just db-reset

# Run migrations
just migrate-up
just migrate-down

# Generate database models from schema
just models
```

### Docker Operations
```bash
# Build Docker image
just docker

# Build and push to registry
just docker-push

# Run complete stack (2 servers + scheduler + postgres)
docker compose up
```

## Architecture

### Core Components

1. **cmd/parsec/**: CLI entry points
   - `cmd.go`: Main CLI application with global flags
   - `cmd_server.go`: Server command implementation
   - `cmd_scheduler.go`: Scheduler command implementation

2. **pkg/server/**: HTTP server implementing the parsec API
   - Handles `/provide` and `/retrieve/{cid}` endpoints per `server.yaml` spec
   - Manages libp2p DHT interactions and timing measurements
   - Reports results to PostgreSQL database

3. **pkg/dht/**: libp2p DHT abstraction layer
   - Host setup and configuration
   - DHT vs IPNI routing implementations
   - Performance metrics collection

4. **pkg/db/**: Database client and models
   - PostgreSQL connection management
   - SQLBoiler-generated models in `pkg/models/`
   - Database schema migrations in `pkg/db/migrations/`

5. **pkg/config/**: Configuration management
   - Global, Server, and Scheduler configs
   - ECS metadata handling for AWS deployments
   - Environment variable mapping

### Database Schema

Key tables:
- `nodes_ecs`: Server node registry with metadata and heartbeat
- `provides_ecs`: Publication timing results
- `retrievals_ecs`: Retrieval timing results
- `schedulers_ecs`: Scheduler metadata

### Fleet System

Fleets are logical groups of servers with different configurations:
- `default`: Standard go-libp2p-kad-dht configuration
- `fullrt`: Accelerated DHT client configuration
- `optprov`: Optimistic provide configuration
- `cid.contact`: IPNI integration

### API Specification

The server HTTP API is defined in `server.yaml` (OpenAPI 3.0):
- `POST /provide`: Publish provider records for content
- `POST /retrieve/{cid}`: Look up provider records
- `GET /readiness`: Health check endpoint

## Configuration

### Environment Variables

Global configuration (prefix `PARSEC_`):
- Database: `PARSEC_DATABASE_HOST`, `PARSEC_DATABASE_PORT`, etc.
- Telemetry: `PARSEC_TELEMETRY_HOST`, `PARSEC_TELEMETRY_PORT`
- Debug: `PARSEC_DEBUG`, `PARSEC_LOG_LEVEL`

Server configuration (prefix `PARSEC_SERVER_`):
- `PARSEC_SERVER_SERVER_HOST/PORT`: HTTP API binding
- `PARSEC_SERVER_PEER_PORT`: libp2p peer port
- `PARSEC_SERVER_FLEET`: Fleet identifier
- `PARSEC_SERVER_FULLRT`: Enable full routing table

Scheduler configuration (prefix `PARSEC_SCHEDULER_`):
- `PARSEC_SCHEDULER_FLEETS`: Comma-separated fleet names
- `PARSEC_SCHEDULER_ROUTING`: DHT or IPNI routing

### AWS ECS Integration

Supports AWS ECS deployment with automatic metadata discovery:
- `ECS_CONTAINER_METADATA_URI_V4`: Metadata endpoint
- `AWS_REGION`: Deployment region
- VPC peering for cross-region connectivity

## Testing and Development

### Local Development
```bash
# Start local database
just database

# Run database setup
just db-reset

# Start a server
go run cmd/parsec/main.go server --fleet=local

# Start a scheduler
go run cmd/parsec/main.go scheduler --fleets=local
```

### Test Structure
- Unit tests for utilities in `pkg/util/`
- Integration tests require database setup
- Use `just test` to run the full test suite

## Code Organization

### Key Packages
- `pkg/util/`: Shared utilities (content generation, formatting)
- `pkg/firehose/`: AWS Kinesis Firehose integration for telemetry
- `pkg/metrics/`: Prometheus metrics collection
- `js-libp2p/`: JavaScript libp2p implementation (separate component)

### Dependencies
- **libp2p**: P2P networking and DHT implementation
- **SQLBoiler**: Database ORM with code generation
- **CLI v2**: Command-line interface framework
- **Prometheus**: Metrics collection and export
- **PostgreSQL**: Primary data store

### Database Model Generation

Models are auto-generated using SQLBoiler:
- Configuration in `sqlboiler.toml`
- Excludes certain tables from generation (see blacklist)
- Custom table aliases for ECS tables
- Run `just models` to regenerate after schema changes