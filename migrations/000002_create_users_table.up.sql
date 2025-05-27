CREATE TABLE IF NOT EXISTS users
(
    id            uuid PRIMARY KEY                     DEFAULT gen_random_uuid(),
    email         citext UNIQUE               NOT NULL,
    password_hash bytea                       NOT NULL,
    created_at    timestamp(0) WITH TIME ZONE NOT NULL DEFAULT now()
);