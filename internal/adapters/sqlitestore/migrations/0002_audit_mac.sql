-- The audit chain becomes HMAC-keyed (ADR-0025): each entry stores a keyed MAC
-- rather than a bare SHA-256 hash. Rename the columns to match; existing rows'
-- values remain valid links (the chaining is unchanged, only the function that
-- produced them is now keyed for entries written henceforth).

ALTER TABLE audit_log RENAME COLUMN prev_hash TO prev_mac;
ALTER TABLE audit_log RENAME COLUMN hash TO mac;
