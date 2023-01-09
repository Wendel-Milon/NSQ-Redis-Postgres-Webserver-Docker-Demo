-- Table: public.users

-- DROP TABLE IF EXISTS public.users;

CREATE TABLE IF NOT EXISTS public.users
(
    userid text COLLATE pg_catalog."default" NOT NULL,
    passwd bytea NOT NULL,
    CONSTRAINT "Users_pkey" PRIMARY KEY (userid)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.users
    OWNER to postgres;