-- +goose Up

-- Sessions previously stored the plaintext random token in session_hash.
-- The application now stores only the SHA-256 hash. Existing rows can no
-- longer be validated, so we clear them; users will be prompted to log in
-- again, which is expected for a credential-format change.
TRUNCATE TABLE sessions;

-- +goose Down
-- No rollback path: cannot recover plaintext from hashes.
