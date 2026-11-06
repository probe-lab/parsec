# parsec

ProbeLab repository. Follow the `probelab` skill: it holds the company rules. Load it before you start any task here.

`parsec` measures `PUT` and `GET` performance of the IPFS Amino DHT, other libp2p-kad-dht networks, and IPNI. A **server** is a libp2p peer with an HTTP API (`server.yaml`, OpenAPI 3.0). A **scheduler** tells one server to publish content and the others to retrieve it, then stores the timings in PostgreSQL. A **fleet** is a group of servers with one configuration: `default`, `fullrt`, `optprov`, `cid.contact`.

## Layout

- `cmd/parsec/` — CLI: `cmd.go` (global flags), `cmd_server*.go` (`server dht`, `server ipni`), `cmd_scheduler.go`, `metrics.go`
- `pkg/server/` — HTTP API (`/provide`, `/retrieve/{cid}`, `/readiness`), libp2p host, DHT and IPNI measurements, metrics
- `pkg/db/` — PostgreSQL client and migrations (`pkg/db/migrations/`)
- `pkg/models/` — SQLBoiler output; never edit by hand, run `just models` (config in `sqlboiler.toml`)
- `pkg/config/` — ECS metadata handling
- `pkg/firehose/` — AWS Kinesis Firehose submitter for telemetry events
- `pkg/util/` — content generation and formatting
- `js-libp2p/` — separate JavaScript server implementation with its own Dockerfile
- `server.yaml` — OpenAPI spec of the server API

## Commands

- `just build`, `just test`, `just lint`, `just format`, `just ci` (the checks CI runs)
- `just tools` installs `migrate` and `sqlboiler`; `just database` starts PostgreSQL in Docker
- `just db-reset` runs migrations down and up, then regenerates models; `just migrate-up`, `just migrate-down`, `just models`
- `just docker` builds the image; `just docker-push` pushes it to ECR
- `docker compose up` starts two servers, a scheduler and PostgreSQL
- Run locally: `go run ./cmd/parsec server dht --fleet=local` and `go run ./cmd/parsec scheduler --fleets=local`
- Integration tests need a database. Most packages have no tests yet.

## Conventions

- Configuration comes from flags and `PARSEC_*` environment variables; `.env.example` lists all of them.
- Changing the schema: add a migration in `pkg/db/migrations/`, then run `just models`.
- Metrics use the Prometheus client directly, not go-commons telemetry. They are served on `PARSEC_TELEMETRY_HOST:PORT` (default `localhost:6666`).
- `staticcheck.conf` disables checks for known existing findings. Remove an entry when you fix its finding.
- The `server` commands write to `nodes_ecs`, `provides_ecs`, `retrievals_ecs`; the scheduler writes `schedulers_ecs`. Other implementations of the server API must follow the table contract in the README.

## Related repositories

- `probelab-infra` — Terraform for the ECS deployment (`tf/modules/parsec_server`, `tf/modules/parsec_scheduler`, `tf/parsec_*.tf`) and the parsec database (`db/cmi_parsec.sql`)

## Owner

@probe-lab/probelab-core
