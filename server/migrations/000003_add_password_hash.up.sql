-- "Lock the doors" (milestone #2): accounts now require a password.
--
-- The pre-auth accounts were password-less LAN/dev stubs with no way to set
-- a credential, so we clear them rather than carry NULL hashes forward. This
-- is a deliberate pre-public reset: TRUNCATE ... CASCADE also clears
-- inventory_items (FK ON DELETE CASCADE) and anything else keyed to a user.
-- Players (e.g. "Ben") simply re-register once.
TRUNCATE TABLE users CASCADE;

-- NOT NULL is safe now that the table is empty: every future row arrives via
-- registration, which always supplies a hash. The format is the argon2id PHC
-- string produced by auth.HashPassword.
ALTER TABLE users ADD COLUMN password_hash text NOT NULL;
