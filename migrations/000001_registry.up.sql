BEGIN;

CREATE TABLE registry_sync_state (
    chain_id BIGINT NOT NULL CHECK (chain_id > 0),
    platform TEXT NOT NULL CHECK (length(platform) BETWEEN 1 AND 64),
    registry_address BYTEA NOT NULL CHECK (octet_length(registry_address) = 20),
    start_block BIGINT NOT NULL CHECK (start_block > 0),
    decoder_version INTEGER NOT NULL CHECK (decoder_version > 0),
    last_scanned_block BIGINT,
    last_scanned_hash BYTEA,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chain_id, platform, registry_address),
    CHECK ((last_scanned_block IS NULL AND last_scanned_hash IS NULL)
        OR (last_scanned_block IS NOT NULL AND last_scanned_block >= start_block
            AND last_scanned_hash IS NOT NULL AND octet_length(last_scanned_hash) = 32))
);

CREATE TABLE registry_checkpoints (
    chain_id BIGINT NOT NULL,
    platform TEXT NOT NULL,
    registry_address BYTEA NOT NULL,
    block_number BIGINT NOT NULL CHECK (block_number >= 0),
    block_hash BYTEA NOT NULL CHECK (octet_length(block_hash) = 32),
    PRIMARY KEY (chain_id, platform, registry_address, block_number),
    FOREIGN KEY (chain_id, platform, registry_address)
        REFERENCES registry_sync_state ON DELETE CASCADE
);

-- Sparse registry events only, not all chain logs or per-block price snapshots.
CREATE TABLE registry_events (
    chain_id BIGINT NOT NULL,
    platform TEXT NOT NULL,
    registry_address BYTEA NOT NULL,
    block_number BIGINT NOT NULL CHECK (block_number >= 0),
    block_hash BYTEA NOT NULL CHECK (octet_length(block_hash) = 32),
    transaction_hash BYTEA NOT NULL CHECK (octet_length(transaction_hash) = 32),
    transaction_index BIGINT NOT NULL CHECK (transaction_index >= 0),
    log_index BIGINT NOT NULL CHECK (log_index >= 0),
    event_kind TEXT NOT NULL CHECK (event_kind IN ('vault_registered', 'gauge_shutdown', 'gauge_unshutdown')),
    gauge_address BYTEA NOT NULL CHECK (octet_length(gauge_address) = 20),
    vault_address BYTEA,
    asset_address BYTEA,
    reward_receiver BYTEA,
    protocol_id BYTEA,
    PRIMARY KEY (chain_id, platform, registry_address, block_number, log_index),
    FOREIGN KEY (chain_id, platform, registry_address)
        REFERENCES registry_sync_state ON DELETE CASCADE,
    CHECK ((event_kind = 'vault_registered'
        AND vault_address IS NOT NULL AND octet_length(vault_address) = 20
        AND asset_address IS NOT NULL AND octet_length(asset_address) = 20
        AND reward_receiver IS NOT NULL AND octet_length(reward_receiver) = 20
        AND protocol_id IS NOT NULL AND octet_length(protocol_id) = 4)
      OR (event_kind <> 'vault_registered' AND vault_address IS NULL
        AND asset_address IS NULL AND reward_receiver IS NULL AND protocol_id IS NULL))
);
CREATE INDEX registry_latest_vault ON registry_events
    (chain_id, platform, registry_address, gauge_address, block_number DESC, transaction_index DESC, log_index DESC)
    WHERE event_kind = 'vault_registered';

COMMIT;
