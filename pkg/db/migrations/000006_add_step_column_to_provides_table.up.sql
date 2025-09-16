BEGIN;

-- Add the step column as nullable first
ALTER TABLE provides_ecs ADD COLUMN step TEXT;

-- Update existing data based on fleet mapping from nodes_ecs
UPDATE provides_ecs
SET step = CASE
    WHEN n.fleet = 'cid.contact' THEN 'availability'
    ELSE 'provide'
END
FROM nodes_ecs n
WHERE provides_ecs.node_id = n.id;

-- Make the column NOT NULL
ALTER TABLE provides_ecs ALTER COLUMN step SET NOT NULL;

COMMIT;