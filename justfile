# Variables
REPO_SERVER := "019120760881.dkr.ecr.us-east-1.amazonaws.com"
GIT_DIRTY := `git diff --quiet || echo '-dirty'`
GIT_SHA := `git rev-parse --short HEAD`
GIT_TAG := GIT_SHA + GIT_DIRTY
VERSION := `cat version`

# The lint tools, pinned here so a person and CI run the same versions.
gofumpt := "go run mvdan.cc/gofumpt@v0.12.0"
staticcheck := "go run honnef.co/go/tools/cmd/staticcheck@2026.2.2"

# List the recipes
default:
    @just --list

# Build Docker image
docker:
    docker build -t "{{REPO_SERVER}}/probelab:parsec-sha{{GIT_TAG}}" .

# Build and push Docker image
docker-push: docker
    docker push "{{REPO_SERVER}}/probelab:parsec-sha{{GIT_TAG}}"
    docker rmi "{{REPO_SERVER}}/probelab:parsec-sha{{GIT_TAG}}"

# Run the tests
test *args:
    go test ./... {{ args }}

# Build the binary
build:
    go build -ldflags "-X main.RawVersion={{VERSION}} -X main.ShortCommit={{GIT_TAG}}" -o dist/parsec github.com/probe-lab/parsec/cmd/parsec

# Check formatting, go fix, vet, and staticcheck without changing files
lint:
    @test -z "$({{ gofumpt }} -l .)" || { echo "not formatted, run just format:"; {{ gofumpt }} -l .; exit 1; }
    @d=$(go fix -diff ./...) && test -z "$d" || { echo "$d"; echo "go fix changes pending, run just format"; exit 1; }
    go vet ./...
    {{ staticcheck }} ./...

# Format the code and apply go fix in place
format:
    {{ gofumpt }} -w .
    go fix ./...

# Run everything CI runs
ci: lint build test

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