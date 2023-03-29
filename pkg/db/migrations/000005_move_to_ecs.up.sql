BEGIN;

CREATE TABLE schedulers_ecs
(
    id           INT GENERATED ALWAYS AS IDENTITY,
    fleets       TEXT[]      NOT NULL,
    dependencies JSONB       NOT NULL,

    created_at   TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE nodes_ecs
(
    id             INT GENERATED ALWAYS AS IDENTITY,
    cpu            INT         NOT NULL,
    memory         INT         NOT NULL,
    peer_id        TEXT        NOT NULL,
    region         TEXT        NOT NULL,
    cmd            TEXT        NOT NULL,
    fleet          TEXT        NOT NULL,
    dependencies   JSONB       NOT NULL,
    ip_address     INET        NOT NULL,
    server_port    SMALLINT    NOT NULL,
    peer_port      SMALLINT    NOT NULL,
    last_heartbeat TIMESTAMPTZ,
    offline_since  TIMESTAMPTZ,

    created_at     TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE retrievals_ecs
(
    id           INT GENERATED ALWAYS AS IDENTITY,
    scheduler_id INT         NOT NULL,
    node_id      INT         NOT NULL,
    rt_size      INT         NOT NULL,
    duration     FLOAT       NOT NULL,
    cid          TEXT        NOT NULL,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_retrievals_ecs_node_id
        FOREIGN KEY (node_id)
            REFERENCES nodes_ecs (id)
            ON DELETE CASCADE,

    CONSTRAINT fk_retrievals_ecs_scheduler_id
        FOREIGN KEY (scheduler_id)
            REFERENCES schedulers_ecs (id)
            ON DELETE CASCADE,

    PRIMARY KEY (id)
);

CREATE TABLE provides_ecs
(
    id           INT GENERATED ALWAYS AS IDENTITY,
    scheduler_id INT         NOT NULL,
    node_id      INT         NOT NULL,
    rt_size      INT         NOT NULL,
    duration     FLOAT       NOT NULL,
    cid          TEXT        NOT NULL,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_provides_ecs_node_id
        FOREIGN KEY (node_id)
            REFERENCES nodes_ecs (id)
            ON DELETE CASCADE,

    CONSTRAINT fk_provides_ecs_scheduler_id
        FOREIGN KEY (scheduler_id)
            REFERENCES schedulers_ecs (id)
            ON DELETE CASCADE,

    PRIMARY KEY (id)
);

CREATE INDEX idx_provides_ecs_created_at ON provides_ecs (created_at);

CREATE INDEX idx_retrievals_ecs_created_at ON retrievals_ecs (created_at);

COMMIT;
