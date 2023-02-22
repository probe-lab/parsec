FROM golang:1.19 AS builder

WORKDIR /build

RUN GOARCH=amd64 GOOS=linux go install github.com/guseggert/clustertest/cmd/agent@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN GOARCH=amd64 GOOS=linux go build -o parsec github.com/dennis-tra/parsec

# Create lightweight container
FROM alpine:latest

RUN adduser -D -H parsec
WORKDIR /home/parsec
USER parsec

COPY --from=builder /build/parsec /usr/local/bin/parsec
COPY --from=builder /go/bin/linux_amd64/agent /home/parsec/nodeagent

CMD parsec schedule