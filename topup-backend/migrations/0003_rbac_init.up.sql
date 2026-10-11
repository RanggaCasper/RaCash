-- 0003_rbac_init: tabel modul rbac (database.dbml § MODUL: rbac)
-- Role sistem & katalog permission di-seed dari kode (cmd/migrate, RBAC-01).

CREATE TYPE role_scope AS ENUM ('internal', 'customer');

CREATE TABLE roles (
    id          uuid PRIMARY KEY,
    code        varchar(64)  NOT NULL,
    name        varchar(100) NOT NULL,
    description text,
    scope       role_scope   NOT NULL,
    is_system   boolean      NOT NULL DEFAULT false,
    created_by  uuid REFERENCES users (id),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT roles_code_uq UNIQUE (code),
    CONSTRAINT roles_code_chk CHECK (code ~ '^[a-z0-9_]+$')
);

CREATE TABLE permissions (
    id           uuid PRIMARY KEY,
    code         varchar(100) NOT NULL,
    resource     varchar(50)  NOT NULL,
    action       varchar(50)  NOT NULL,
    description  varchar(255) NOT NULL,
    is_sensitive boolean      NOT NULL DEFAULT false,
    created_at   timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT permissions_code_uq UNIQUE (code),
    CONSTRAINT permissions_code_chk CHECK (code = resource || ':' || action)
);
CREATE INDEX permissions_resource_idx ON permissions (resource);

CREATE TABLE role_permissions (
    role_id       uuid        NOT NULL REFERENCES roles (id),
    permission_id uuid        NOT NULL REFERENCES permissions (id),
    granted_by    uuid REFERENCES users (id),
    granted_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);
CREATE INDEX role_permissions_permission_id_idx ON role_permissions (permission_id);

CREATE TABLE user_roles (
    user_id     uuid        NOT NULL REFERENCES users (id),
    role_id     uuid        NOT NULL REFERENCES roles (id),
    assigned_by uuid REFERENCES users (id),
    assigned_at timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX user_roles_role_id_idx ON user_roles (role_id);
