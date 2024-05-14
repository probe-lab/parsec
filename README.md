# `parsec`

`parsec` is a DHT lookup performance measurement tool. It specifically measures the `PUT` and `GET` performance of the
IPFS public DHT but could also be configured to measure
other [libp2p-kad-dht](https://github.com/libp2p/specs/blob/master/kad-dht/README.md) networks.
The setup is split into two components: a scheduler and a server.

The server is just a normal libp2p peer that supports and participates in the public IPFS DHT and exposes a [lean HTTP
API](./server.yaml) that allows the scheduler to issue publication and retrieval operations. Currently, in [ProbeLab's](https://probelab.io/tools/parsec/)
deployment, the scheduler goes around all seven server nodes, instructs one to publish provider records for a random data
blob and asks the other six to look them up. All seven servers take timing measurements about the publication or retrieval latencies and
report back the results to the scheduler. The scheduler then tracks this information in a database for later analysis.

## Table of Contents

- [`parsec`](#parsec)
  - [Table of Contents](#table-of-contents)
  - [Concepts](#concepts)
  - [Running](#running)
  - [Implementing a Server](#implementing-a-server)
    - [Expose the HTTP API](#expose-the-http-api)
    - [Node Information](#node-information)
    - [Heartbeat](#heartbeat)
    - [Optional: Prometheus Metrics](#optional-prometheus-metrics)
    - [`ECS_CONTAINER_METADATA_URI_V4` response:](#ecs_container_metadata_uri_v4-response)
  - [Maintainers](#maintainers)
  - [Contributing](#contributing)
  - [License](#license)


## Concepts

Next to the concept of servers and schedulers there's the concept of a `fleet`. A fleet is a set of server nodes that
have a common configuration. For example, we are running three different fleets with seven nodes each (in different regions): 1) `default` 2) `optprov` 3) `fullrt`.
Each of these three fleets are configured differently. The `default` fleet uses the default configuration in the `go-libp2p-kad-dht` repository, the `optprov` fleet uses the optimistic provide configuration to publish data into the DHT, and the `fullrt` fleet uses the accelerated DHT client.

Schedulers are then configured to interface with any combination of fleets. Right now, we have one scheduler for each fleet. As said above, it asks one node to publish content, then instructs the others to find the provider records, and then repeats the process with the next peer. However,
we could configure a scheduler that does the same thing but with nodes from multiple fleets e.g., `default`+`fullrt` to check if content that's published with one implementation is reachable with another one.

## Running

You can run

```shell
docker compose up
```

to start two servers and one scheduler and see them interact.

## Implementing a Server

Right now, the server component is implemented in Go and uses the [go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht) implementation.
It consequently measures the Go implementations performance. However, other implementations exist that support the DHT protocol. These
implementations can be easily integrated with this measurement infrastructure. They just need to behave as a parsec server.
The existing schedulers can then be reused.

Here are the things that a new implementation would need to do:

1. Expose an HTTP interface with three endpoints
2. Upon startup write general information about the node configuration to a postgres database.
3. Regularly refresh the heartbeat field in the database.

### Expose the HTTP API

You can find the OpenAPI specification in the [`./server.yaml`](./server.yaml).

### Node Information

The new server would need to initialize a postgres client. The default environment variables to configure the client are as follows:

```env
PARSEC_DATABASE_HOST
PARSEC_DATABASE_PORT
PARSEC_DATABASE_NAME
PARSEC_DATABASE_PASSWORD
PARSEC_DATABASE_USER
PARSEC_DATABASE_SSL_MODE
```

Then upon startup, the server needs to write a row into the `nodes_ecs` table. The definition looks like this:

```sql
CREATE TABLE nodes_ecs
(
    -- auto generated, doesn't need to be set manually
    id             INT GENERATED ALWAYS AS IDENTITY,
    -- available CPUs
    cpu            INT         NOT NULL,
    -- available memory rounded to the nearest MB
    memory         INT         NOT NULL,
    -- the peer ID of the libp2p host
    peer_id        TEXT        NOT NULL,
    -- in which region does this server/node run? Given via the AWS_REGION environment var
    region         TEXT        NOT NULL,
    -- os.Args - with which arguments was this server run?
    cmd            TEXT        NOT NULL,
    -- a fleet identifier (see section `Concepts` above)
    fleet          TEXT        NOT NULL,
    -- a JSON document with no enforced schema. Could be anything really. It's intended
    -- to give information about the exact dependencies, and especially kad-dht implementation
    -- that the server uses. 
    dependencies   JSONB       NOT NULL,
    -- the private IP address of the server. The scheduler will query this table and use this
    -- ip address to contact the HTTP API. In the ECS context it's provided via an environment
    -- variable called `ECS_CONTAINER_METADATA_URI_V4`. https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html
    -- see below for the JSON structure
    ip_address     INET        NOT NULL,
    -- under which port is the HTTP server reachable
    server_port    SMALLINT    NOT NULL,
    -- which port does the libp2p host use
    peer_port      SMALLINT    NOT NULL,
    -- a timestamp of the last heartbeat
    last_heartbeat TIMESTAMPTZ,
    -- a timestamp since when the node is offline (set from the scheduler if the node is unreachable, e.g., crashed)
    -- but should also be set from server when shutdown gracefully.
    offline_since  TIMESTAMPTZ,
    -- when was this node row created.
    created_at     TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (id)
);
```

This row serves two purposes: 1) it tracks the exact configuration and dependencies of the server and 2) is used for service discovery.
The scheduler will query this table for all nodes that matches the `fleet` that the scheduler is after where the `offline_since` field is null and
the `last_heartbeat` is not null. In AWS, we're using [VPC peering](https://docs.aws.amazon.com/vpc/latest/peering/what-is-vpc-peering.html) between different AWS regions, so we can use private IP addresses for connectivity between the scheduler, servers, and database.

### Heartbeat

The server must update the `node_ecs` row every minute to indicate it's still alive and happy to accept requests.

### Optional: Prometheus Metrics

To expose real time metrics about the publication and retrieval performance the Go server exposes a few prometheus metrics:

```
metric: parsec_durations
type: summary
buckets: 50th, 90th, 95th percentile
maxAge: 24h
labels:
  type: retrieval_ttfpr | provide_duration
  success: true | false
  scheduler: default | optprov | fullrt
```

```
metric: parsec_http_requests_total
type: summary
buckets: 50th, 90th, 95th percentile
maxAge: 24h
labels:
  method: GET | POST | ...
  path: /retrieve | /provide 
  scheduler: default | optprov | fullrt
```

### `ECS_CONTAINER_METADATA_URI_V4` response:

The server can extract the available CPU and Memory from `Limits.CPU` and `Limits.Memory`. Further,
its own private IP address is in the `Networks` array. Look for the first entry with `NetworkMode == awsvpc`
and then just take the first entry of `IPv4Addresses` - that's a good enough heuristic so far.

```json
{
    "DockerId": "ea32192c8553fbff06c9340478a2ff089b2bb5646fb718b4ee206641c9086d66",
    "Name": "curl",
    "DockerName": "ecs-curltest-24-curl-cca48e8dcadd97805600",
    "Image": "111122223333.dkr.ecr.us-west-2.amazonaws.com/curltest:latest",
    "ImageID": "sha256:d691691e9652791a60114e67b365688d20d19940dde7c4736ea30e660d8d3553",
    "Labels": {
        "com.amazonaws.ecs.cluster": "default",
        "com.amazonaws.ecs.container-name": "curl",
        "com.amazonaws.ecs.task-arn": "arn:aws:ecs:us-west-2:111122223333:task/default/8f03e41243824aea923aca126495f665",
        "com.amazonaws.ecs.task-definition-family": "curltest",
        "com.amazonaws.ecs.task-definition-version": "24"
    },
    "DesiredStatus": "RUNNING",
    "KnownStatus": "RUNNING",
    "Limits": {
        "CPU": 10,
        "Memory": 128
    },
    "CreatedAt": "2020-10-02T00:15:07.620912337Z",
    "StartedAt": "2020-10-02T00:15:08.062559351Z",
    "Type": "NORMAL",
    "LogDriver": "awslogs",
    "LogOptions": {
        "awslogs-create-group": "true",
        "awslogs-group": "/ecs/metadata",
        "awslogs-region": "us-west-2",
        "awslogs-stream": "ecs/curl/8f03e41243824aea923aca126495f665"
    },
    "ContainerARN": "arn:aws:ecs:us-west-2:111122223333:container/0206b271-b33f-47ab-86c6-a0ba208a70a9",
    "Networks": [
        {
            "NetworkMode": "awsvpc",
            "IPv4Addresses": [
                "10.0.2.100"
            ],
            "AttachmentIndex": 0,
            "MACAddress": "0e:9e:32:c7:48:85",
            "IPv4SubnetCIDRBlock": "10.0.2.0/24",
            "PrivateDNSName": "ip-10-0-2-100.us-west-2.compute.internal",
            "SubnetGatewayIpv4Address": "10.0.2.1/24"
        }
    ]
}
```

## Maintainers

[@dennis-tra](https://github.com/dennis-tra).

## Contributing

Feel free to dive in! [Open an issue](https://github.com/probe-lab/parsec/issues/new) or submit PRs.

## License

[MIT](LICENSE) © Dennis Trautwein
