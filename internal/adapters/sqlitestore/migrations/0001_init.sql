-- Initial schema (PROJECT-BOOK §7.7, ADR-0007). Forward-only; never edited once
-- shipped. Times are stored as Unix nanoseconds (INTEGER).

CREATE TABLE hosts (
    id                  TEXT    PRIMARY KEY,
    name                TEXT    NOT NULL,
    transport           TEXT    NOT NULL,
    endpoint            TEXT    NOT NULL,
    observe_mode        INTEGER NOT NULL DEFAULT 0,
    key_ref             TEXT,
    last_engine_version TEXT,
    last_api_version    TEXT,
    created_at          INTEGER NOT NULL
);

CREATE TABLE operations (
    id              TEXT    PRIMARY KEY,
    host_id         TEXT    NOT NULL REFERENCES hosts(id),
    kind            TEXT    NOT NULL,
    target          TEXT    NOT NULL,
    option_set      TEXT    NOT NULL DEFAULT '{}',
    result          TEXT    NOT NULL DEFAULT '',
    bytes_reclaimed INTEGER NOT NULL DEFAULT 0,
    started_at      INTEGER NOT NULL,
    ended_at        INTEGER
);

CREATE INDEX idx_operations_host ON operations(host_id, started_at);

CREATE TABLE prune_impacts (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    operation_id      TEXT    NOT NULL REFERENCES operations(id),
    category          TEXT    NOT NULL,
    object_count      INTEGER NOT NULL,
    reclaimable_bytes INTEGER NOT NULL
);

CREATE INDEX idx_prune_impacts_operation ON prune_impacts(operation_id);

CREATE TABLE resource_samples (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    host_id      TEXT    NOT NULL REFERENCES hosts(id),
    container_id TEXT    NOT NULL,
    at           INTEGER NOT NULL,
    cpu_pct      REAL    NOT NULL,
    mem_bytes    INTEGER NOT NULL,
    net_rx       INTEGER NOT NULL,
    net_tx       INTEGER NOT NULL,
    blk_read     INTEGER NOT NULL,
    blk_write    INTEGER NOT NULL
);

CREATE INDEX idx_resource_samples_window ON resource_samples(host_id, container_id, at);

CREATE TABLE sealed_values (
    id           TEXT PRIMARY KEY,
    scope        TEXT NOT NULL,
    nonce        BLOB NOT NULL,
    sealed_bytes BLOB NOT NULL
);

-- The audit log is independent of hosts (no foreign key): it must remain
-- durable and verifiable even after a host profile is deleted (ADR-0010).
CREATE TABLE audit_log (
    seq       INTEGER PRIMARY KEY,
    at        INTEGER NOT NULL,
    action    TEXT    NOT NULL,
    host_id   TEXT,
    subject   TEXT    NOT NULL DEFAULT '',
    detail    TEXT    NOT NULL DEFAULT 'null',
    prev_hash TEXT    NOT NULL DEFAULT '',
    hash      TEXT    NOT NULL
);
