CREATE TABLE inventory_items (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_type  text NOT NULL,
    quantity   integer NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_type)
);
