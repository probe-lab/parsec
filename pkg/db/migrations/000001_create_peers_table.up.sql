BEGIN;

-- The `peers` table keeps track of all peers ever seen
CREATE TABLE peers
(
    id            INT GENERATED ALWAYS AS IDENTITY,
    multi_hash    TEXT        NOT NULL,
    agent_version TEXT,
    protocols     TEXT[],
    updated_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL,

    CONSTRAINT uq_peers_multi_hash UNIQUE (multi_hash),

    PRIMARY KEY (id)
);

CREATE OR REPLACE FUNCTION upsert_peer(
    new_multi_hash TEXT,
    new_agent_version TEXT DEFAULT NULL,
    new_protocols TEXT[] DEFAULT NULL,
    new_created_at TIMESTAMPTZ DEFAULT NOW()
) RETURNS INT AS
$upsert_peer$
WITH sel AS (--
    SELECT id, multi_hash, agent_version, protocols
    FROM peers
    WHERE multi_hash = new_multi_hash
), ups AS (--
     INSERT INTO peers AS p (multi_hash, agent_version, protocols, updated_at, created_at)
     SELECT new_multi_hash, new_agent_version, new_protocols, new_created_at, new_created_at
     WHERE NOT EXISTS(SELECT NULL FROM sel)
     ON CONFLICT ON CONSTRAINT uq_peers_multi_hash DO UPDATE
     SET multi_hash = EXCLUDED.multi_hash,
         agent_version = coalesce(EXCLUDED.agent_version, p.agent_version),
         protocols = coalesce(EXCLUDED.protocols, p.protocols)
     RETURNING id, multi_hash
), upd AS (--
     UPDATE peers
     SET agent_version = coalesce(new_agent_version, agent_version),
         protocols = coalesce(new_protocols, protocols),
         updated_at = new_created_at
     WHERE id = (SELECT id FROM sel) AND (
                 coalesce(agent_version, -1) != coalesce(new_agent_version, -1) OR
                 coalesce(protocols, -1) != coalesce(new_protocols, -1)
         )
     RETURNING peers.id--
)
SELECT id
FROM sel
UNION
SELECT id
FROM ups
UNION
SELECT id
FROM upd;
$upsert_peer$ LANGUAGE sql;

COMMIT;
