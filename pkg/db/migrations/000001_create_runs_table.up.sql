BEGIN;

CREATE TABLE runs
(
    id           INT GENERATED ALWAYS AS IDENTITY,

    dependencies JSONB       NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL,
    finished_at  TIMESTAMPTZ CHECK ( finished_at > started_at ),
    updated_at   TIMESTAMPTZ NOT NULL CHECK ( updated_at >= created_at ),
    created_at   TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (id)
);

COMMIT;
