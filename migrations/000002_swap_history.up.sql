BEGIN;
-- Same database/pool; independent keys and progress from Controller registrations.
CREATE TABLE swap_streams (
 chain_id BIGINT NOT NULL CHECK (chain_id>0),
 platform TEXT NOT NULL CHECK (length(platform) BETWEEN 1 AND 64),
 pool_address BYTEA NOT NULL CHECK (octet_length(pool_address)=20),
 event_topic BYTEA NOT NULL CHECK (octet_length(event_topic)=32),
 decoder_version INTEGER NOT NULL CHECK (decoder_version>0),
 last_saved_block BIGINT,
 last_saved_hash BYTEA,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(chain_id,platform,pool_address,event_topic),
 CHECK ((last_saved_block IS NULL AND last_saved_hash IS NULL) OR
  (last_saved_block IS NOT NULL AND last_saved_block>=1 AND last_saved_hash IS NOT NULL AND octet_length(last_saved_hash)=32))
);
-- Normalized disjoint intervals. Empty successfully scanned ranges also persist.
CREATE TABLE swap_scan_ranges (
 chain_id BIGINT NOT NULL, platform TEXT NOT NULL, pool_address BYTEA NOT NULL, event_topic BYTEA NOT NULL,
 from_block BIGINT NOT NULL CHECK(from_block>0),
 through_block BIGINT NOT NULL CHECK(through_block>=from_block),
 through_hash BYTEA NOT NULL CHECK(octet_length(through_hash)=32),
 PRIMARY KEY(chain_id,platform,pool_address,event_topic,from_block),
 FOREIGN KEY(chain_id,platform,pool_address,event_topic) REFERENCES swap_streams ON DELETE CASCADE
);
CREATE INDEX swap_range_end ON swap_scan_ranges(chain_id,platform,pool_address,event_topic,through_block);
CREATE TABLE swap_events (
 chain_id BIGINT NOT NULL, platform TEXT NOT NULL, pool_address BYTEA NOT NULL, event_topic BYTEA NOT NULL,
 block_number BIGINT NOT NULL CHECK(block_number>0),
 block_hash BYTEA NOT NULL CHECK(octet_length(block_hash)=32),
 transaction_hash BYTEA NOT NULL CHECK(octet_length(transaction_hash)=32),
 transaction_index BIGINT NOT NULL CHECK(transaction_index>=0),
 log_index BIGINT NOT NULL CHECK(log_index>=0),
 buyer_address BYTEA NOT NULL CHECK(octet_length(buyer_address)=20),
 event_data BYTEA NOT NULL CHECK(octet_length(event_data)=128),
 PRIMARY KEY(chain_id,platform,pool_address,event_topic,block_number,log_index),
 FOREIGN KEY(chain_id,platform,pool_address,event_topic) REFERENCES swap_streams ON DELETE CASCADE
);
COMMIT;
