BEGIN;

ALTER TABLE runs
    RENAME TO clustertest_runs;

ALTER TABLE nodes
    RENAME TO clustertest_nodes;

ALTER TABLE clustertest_nodes
    RENAME CONSTRAINT fk_nodes_run_id TO fk_clustertest_nodes_run_id;

ALTER TABLE retrievals
    RENAME TO clustertest_retrievals;

ALTER TABLE clustertest_retrievals
    RENAME CONSTRAINT fk_retrievals_node TO fk_clustertest_retrievals_node_id;

ALTER INDEX idx_retrievals_created_at RENAME TO idx_clustertest_retrievals_created_at;

ALTER TABLE provides
    RENAME TO clustertest_provides;

ALTER TABLE clustertest_provides
    RENAME CONSTRAINT fk_provides_node TO fk_clustertest_provides_node_id;

ALTER INDEX idx_provides_created_at RENAME TO idx_clustertest_provides_created_at;

CREATE TABLE nodes
(
    id             INT GENERATED ALWAYS AS IDENTITY,
    cpu            INT         NOT NULL,
    memory         INT         NOT NULL,
    peer_id        TEXT        NOT NULL,
    region         TEXT        NOT NULL,
    cmd            TEXT        NOT NULL,
    tags           TEXT[]      NOT NULL,
    dependencies   JSONB       NOT NULL,
    ip_address     INET        NOT NULL,
    server_port    SMALLINT    NOT NULL,
    peer_port      SMALLINT    NOT NULL,
    last_heartbeat TIMESTAMPTZ,
    offline_since  TIMESTAMPTZ,

    created_at     TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE retrievals
(
    id         INT GENERATED ALWAYS AS IDENTITY,
    node_id    INT         NOT NULL,
    rt_size    INT         NOT NULL,
    duration   FLOAT       NOT NULL,
    cid        TEXT        NOT NULL,
    error      TEXT,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_retrievals_node_id
        FOREIGN KEY (node_id)
            REFERENCES nodes (id),

    PRIMARY KEY (id)
);

CREATE TABLE provides
(
    id         INT GENERATED ALWAYS AS IDENTITY,
    node_id    INT         NOT NULL,
    rt_size    INT         NOT NULL,
    duration   FLOAT       NOT NULL,
    cid        TEXT        NOT NULL,
    error      TEXT,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_provides_node_id
        FOREIGN KEY (node_id)
            REFERENCES nodes (id),

    PRIMARY KEY (id)
);

CREATE INDEX idx_provides_created_at ON provides (created_at);

CREATE INDEX idx_retrievals_created_at ON retrievals (created_at);

COMMIT;
