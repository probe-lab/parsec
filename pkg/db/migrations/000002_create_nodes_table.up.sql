BEGIN;

CREATE TABLE nodes
(
    id            INT GENERATED ALWAYS AS IDENTITY,
    peer_id       TEXT        NOT NULL,
    run_id        INT         NOT NULL,
    region        TEXT        NOT NULL,
    instance_type TEXT        NOT NULL,
    dependencies  JSONB       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL,

    CONSTRAINT fk_nodes_run_id FOREIGN KEY (run_id) REFERENCES runs (id) ON DELETE SET NULL,

    PRIMARY KEY (id)
);

COMMIT;