REPO_SERVER=019120760881.dkr.ecr.us-east-1.amazonaws.com

GIT_DIRTY := $(shell git diff --quiet || echo '-dirty')
GIT_SHA := $(shell git rev-parse --short HEAD)
GIT_TAG := ${GIT_SHA}${GIT_DIRTY}

VERSION := $(shell cat version)

docker:
	docker build -t "${REPO_SERVER}/probelab:parsec-${GIT_TAG}" .

docker-push: docker
	docker push "${REPO_SERVER}/probelab:parsec-${GIT_TAG}"
	docker rmi "${REPO_SERVER}/probelab:parsec-${GIT_TAG}"

test:
	go test ./...

build:
	go build -ldflags "-X main.RawVersion=${VERSION} -X main.ShortCommit=${GIT_TAG}" -o dist/parsec github.com/dennis-tra/parsec/cmd/parsec

linux-build:
	GOOS=linux GOARCH=amd64 go build -o dist/parsec cmd/parsec/*

format:
	gofumpt -w -l .

clean:
	rm -r dist || true

tools:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2
	go install github.com/volatiletech/sqlboiler/v4@v4.13.0
	go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql@v4.13.0

db-reset: migrate-down migrate-up models

database:
	docker run --rm -p 5435:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_USER=parsec -e POSTGRES_DB=parsec postgres:14

database-test:
	docker run --rm -p 2345:5432 -e POSTGRES_PASSWORD=password_test -e POSTGRES_USER=parsec_test -e POSTGRES_DB=parsec_test postgres:14

models:
	sqlboiler --no-tests psql

migrate-up:
	migrate -database 'postgres://parsec:password@localhost:5435/parsec?sslmode=disable' -path pkg/db/migrations up

migrate-down:
	migrate -database 'postgres://parsec:password@localhost:5435/parsec?sslmode=disable' -path pkg/db/migrations down

.PHONY: all clean build test format tools models migrate-up migrate-down