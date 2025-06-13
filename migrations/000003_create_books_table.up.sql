CREATE TABLE IF NOT EXISTS books
(
    id         uuid PRIMARY KEY                     DEFAULT gen_random_uuid(),
    user_id    uuid                        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       text                        NOT NULL,
    created_at timestamp(0) WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at timestamp(0) WITH TIME ZONE NOT NULL DEFAULT now(),
    version    int                         NOT NULL DEFAULT 1,
    UNIQUE (user_id, name)
);