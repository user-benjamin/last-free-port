CREATE TABLE users (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username   text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_login timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT username_length CHECK (char_length(username) BETWEEN 2 AND 24)
);
