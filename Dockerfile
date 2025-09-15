FROM golang:1.25 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o parsec github.com/probe-lab/parsec/cmd/parsec

# Create lightweight container
FROM alpine:latest

RUN apk add --update curl && rm -rf /var/cache/apk/*

RUN adduser -D -H parsec
WORKDIR /home/parsec
RUN chown -R parsec:parsec /home/parsec
USER parsec

COPY --from=builder /build/parsec /usr/local/bin/parsec

CMD ["parsec", "scheduler"]
