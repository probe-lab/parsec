BEGIN;

CREATE TABLE measurements
(
    id         INT GENERATED ALWAYS AS IDENTITY,
    node_id    INT         NOT NULL,
    metric     TEXT        NOT NULL,
    value      FLOAT,
    error      TEXT,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_measurements_node
        FOREIGN KEY (node_id)
            REFERENCES nodes (id),

    PRIMARY KEY (id)
);

CREATE INDEX idx_measurements_created_at ON measurements (created_at);

COMMIT;

