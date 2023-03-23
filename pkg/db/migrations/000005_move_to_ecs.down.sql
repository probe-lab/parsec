BEGIN;

DROP TABLE provides;
DROP TABLE retrievals;
DROP TABLE nodes;

ALTER TABLE clustertest_runs
    RENAME TO runs;

ALTER TABLE clustertest_nodes
    RENAME TO nodes;

ALTER TABLE nodes
    RENAME CONSTRAINT fk_clustertest_nodes_run_id TO fk_nodes_run_id;

ALTER TABLE clustertest_retrievals
    RENAME TO retrievals;

ALTER TABLE retrievals
    RENAME CONSTRAINT fk_clustertest_retrievals_node_id TO fk_retrievals_node;

ALTER INDEX idx_clustertest_retrievals_created_at RENAME TO idx_retrievals_created_at;

ALTER TABLE clustertest_provides
    RENAME TO provides;

ALTER TABLE provides
    RENAME CONSTRAINT fk_clustertest_provides_node_id TO fk_provides_node;

ALTER INDEX idx_clustertest_provides_created_at RENAME TO idx_provides_created_at;

COMMIT;
