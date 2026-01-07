CREATE TABLE registration_session
(
    id           UUID PRIMARY KEY   DEFAULT gen_random_uuid(),
    code         TEXT      NOT NULL,
    email        TEXT      NOT NULL UNIQUE,
    code_expires TIMESTAMP NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE login_session
(
    id           UUID PRIMARY KEY   DEFAULT gen_random_uuid(),
    account_id   UUID,
    email        TEXT      NOT NULL UNIQUE,
    code         TEXT      NOT NULL,
    code_expires TIMESTAMP NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE refresh_token_session
(
    id            UUID PRIMARY KEY   DEFAULT gen_random_uuid(),
    account_id    UUID      NOT NULL,
    refresh_token TEXT      NOT NULL,
    expires_at    TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_token_session_refresh_token ON refresh_token_session(refresh_token);
CREATE INDEX idx_refresh_token_session_expires_at ON refresh_token_session(expires_at);
CREATE INDEX idx_login_session_code_expires ON login_session(code_expires);
CREATE INDEX idx_registration_session_code_expires ON registration_session(code_expires);
