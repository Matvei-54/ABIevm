-- Destructive rollback: removes saved registry history and scan progress.
BEGIN;
DROP TABLE registry_events;
DROP TABLE registry_checkpoints;
DROP TABLE registry_sync_state;
COMMIT;
