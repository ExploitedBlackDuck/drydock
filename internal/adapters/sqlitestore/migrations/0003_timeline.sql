-- Host timeline (ADR-0018, §7.12.4): mapped engine events persisted with rolling
-- retention, interleaved on read with audit references. Deliberately has no
-- foreign key to hosts and is entirely separate from audit_log — engine events
-- are untrusted input and never enter the hash-chained audit log.

CREATE TABLE timeline_entries (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    host_id   TEXT    NOT NULL,
    at        INTEGER NOT NULL,
    source    TEXT    NOT NULL,
    kind      TEXT    NOT NULL,
    subject   TEXT    NOT NULL DEFAULT '',
    exit_code INTEGER,
    detail    TEXT    NOT NULL DEFAULT 'null'
);

CREATE INDEX idx_timeline_host ON timeline_entries(host_id, at);
