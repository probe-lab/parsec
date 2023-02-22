efault: all

all: clean build

test:
	go test ./...

build:
	go build -o dist/parsec cmd/parsec/*

linux-build:
	GOOS=linux GOARCH=amd64 go build -o dist/parsec cmd/parsec/*

format:
	gofumpt -w -l .

clean:
	rm -r dist || true

docker:
	docker build -t dennistra/parsec:latest -t dennistra/parsec:`cat version` .

docker-push: docker
	docker push dennistra/parsec:latest dennistra/parsec:`cat version`

tools:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2
	go install github.com/volatiletech/sqlboiler/v4@v4.13.0
	go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql@v4.13.0

db-reset: migrate-down migrate-up models

database:
	docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_USER=parsec -e POSTGRES_DB=parsec postgres:14

database-test:
	docker run --rm -p 2345:5432 -e POSTGRES_PASSWORD=password_test -e POSTGRES_USER=parsec_test -e POSTGRES_DB=parsec_test postgres:14

models:
	sqlboiler --no-tests psql

migrate-up:
	migrate -database 'postgres://parsec:password@localhost:5432/parsec?sslmode=disable' -path pkg/db/migrations up

migrate-down:
	migrate -database 'postgres://parsec:password@localhost:5432/parsec?sslmode=disable' -path pkg/db/migrations down

.PHONY: all clean test format tools models migrate-up migrate-down