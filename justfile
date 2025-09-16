# Variables
REPO_SERVER := "019120760881.dkr.ecr.us-east-1.amazonaws.com"
GIT_DIRTY := `git diff --quiet || echo '-dirty'`
GIT_SHA := `git rev-parse --short HEAD`
GIT_TAG := GIT_SHA + GIT_DIRTY
VERSION := `cat version`

# Default recipe - show available commands
default:
    @just --list

# Build Docker image
docker:
    docker build -t "{{REPO_SERVER}}/probelab:parsec-{{GIT_TAG}}" .

# Build and push Docker image
docker-push: docker
    docker push "{{REPO_SERVER}}/probelab:parsec-{{GIT_TAG}}"
    docker rmi "{{REPO_SERVER}}/probelab:parsec-{{GIT_TAG}}"

# Run tests
test:
    go test ./...

# Build the binary
build:
    go build -ldflags "-X main.RawVersion={{VERSION}} -X main.ShortCommit={{GIT_TAG}}" -o dist/parsec github.com/probe-lab/parsec/cmd/parsec

# Format code
format:
    gofumpt -w -l .

# Clean build artifacts
clean:
    rm -r dist || true

# Install development tools
tools:
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0
    go install github.com/aarondl/sqlboiler/v4@v4.19.5
    go install github.com/aarondl/sqlboiler/v4/drivers/sqlboiler-psql@v4.19.5

# Reset database
db-reset: migrate-down migrate-up models

# Start database container
database:
    docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_USER=parsec -e POSTGRES_DB=parsec postgres:14

# Generate models
models:
    sqlboiler --no-tests psql

# Run database migrations up
migrate-up:
    migrate -database 'postgres://parsec:password@localhost:5432/parsec?sslmode=disable' -path pkg/db/migrations up

# Run database migrations down
migrate-down:
    migrate -database 'postgres://parsec:password@localhost:5432/parsec?sslmode=disable' -path pkg/db/migrations down