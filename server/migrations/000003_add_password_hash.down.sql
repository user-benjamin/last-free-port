-- Reverting drops credentials. The truncate is not undoable; the column is.
ALTER TABLE users DROP COLUMN password_hash;
