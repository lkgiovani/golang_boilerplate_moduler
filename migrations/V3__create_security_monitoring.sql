CREATE TABLE suspicious_activities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_count INTEGER DEFAULT 1,
    details JSONB,
    severity VARCHAR(20) NOT NULL DEFAULT 'LOW',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_activity_type CHECK (activity_type IN (
        'RATE_LIMIT_EXCEEDED',
        'MASS_CREATION',
        'PATTERN_ABUSE',
        'INVALID_DATA_ATTEMPTS',
        'UNAUTHORIZED_ACCESS',
        'SUSPICIOUS_PATTERN',
        'AUTOMATED_BEHAVIOR'
    )),
    CONSTRAINT chk_severity CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL'))
);

CREATE INDEX idx_suspicious_activities_user_id ON suspicious_activities(user_id);
CREATE INDEX idx_suspicious_activities_created_at ON suspicious_activities(created_at);
CREATE INDEX idx_suspicious_activities_severity ON suspicious_activities(severity);
CREATE INDEX idx_suspicious_activities_user_created ON suspicious_activities(user_id, created_at);
CREATE INDEX idx_suspicious_activities_type_severity ON suspicious_activities(activity_type, severity);

CREATE TABLE user_security_blocks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason VARCHAR(500) NOT NULL,
    suspicious_activity_count INTEGER NOT NULL DEFAULT 0,
    blocked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    blocked_until TIMESTAMP,
    unblocked_at TIMESTAMP,
    unblocked_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT
);

CREATE UNIQUE INDEX idx_unique_active_block ON user_security_blocks(user_id) WHERE unblocked_at IS NULL;
CREATE INDEX idx_user_security_blocks_user_id ON user_security_blocks(user_id);
CREATE INDEX idx_user_security_blocks_blocked_at ON user_security_blocks(blocked_at);
CREATE INDEX idx_user_security_blocks_active ON user_security_blocks(user_id) WHERE unblocked_at IS NULL;

COMMENT ON TABLE suspicious_activities IS 'Registra atividades suspeitas de usuarios para analise e bloqueio automatico';
COMMENT ON TABLE user_security_blocks IS 'Registra bloqueios de seguranca temporarios ou permanentes';
COMMENT ON COLUMN user_security_blocks.blocked_until IS 'NULL indica bloqueio permanente, caso contrario e temporario';
