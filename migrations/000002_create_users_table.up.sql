CREATE TABLE IF NOT EXISTS users
(
    id            uuid PRIMARY KEY                     DEFAULT gen_random_uuid(),
    first_name    text                        NOT NULL,
    last_name     text                        NOT NULL,
    email         citext UNIQUE               NOT NULL,
    password_hash bytea                       NOT NULL,
    created_at    timestamp(0) WITH TIME ZONE NOT NULL DEFAULT now()
);