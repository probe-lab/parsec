# `parsec`

`parsec` is a DHT lookup performance measurement tool. It is built on top of [clustertest](https://github.com/guseggert/clustertest)
and provides the infrastructure to deploy probes around the world, allowing you to measure the performance of Kademlia
distributed hash table (DHT) lookups. It works with both Docker and AWS, allowing you to deploy the probes easily and
efficiently. With this tool, you can gather accurate data on the performance of DHT lookups.

## Table of Contents

- [Running](#running)
  - [Docker](#docker)
- [Related Efforts](#related-efforts)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [License](#license)

## Running

### Docker

1. Head over to [clustertest](https://github.com/guseggert/clustertest) and build the `nodeagent` binary. Keep it in the root folder here.
2. On a Mac also build the parsec binary for linux/amd if you want to run it using Docker `GOARCH=amd64 GOOS=linux go build -o parsec cmd/parsec/*.go`
3. Start `parsec` by running `go run cmd/parsec/*.go schedule docker --nodes 4`

## Related Efforts

- [`clustertest`](https://github.com/guseggert/clustertest) - A framework and library for orchestrating clusters of nodes for testing.
- [`clustertest-kubo`](https://github.com/guseggert/clustertest-kubo) - A library for testing Kubo using clustertest.

## Maintainers

[@dennis-tra](https://github.com/dennis-tra).

## Contributing

Feel free to dive in! [Open an issue](https://github.com/dennis-tra/parsec/issues/new) or submit PRs.

## License

[MIT](LICENSE) © Dennis Trautwein