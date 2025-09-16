BEGIN;

-- Drop the index on the step column
DROP INDEX IF EXISTS idx_provides_ecs_step;

-- Drop the step column
ALTER TABLE provides_ecs DROP COLUMN step;

COMMIT;