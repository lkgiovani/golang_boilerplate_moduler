ALTER TABLE suspicious_activities
  ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';

ALTER TABLE user_security_blocks
  ALTER COLUMN blocked_at    TYPE TIMESTAMPTZ USING blocked_at    AT TIME ZONE 'UTC',
  ALTER COLUMN blocked_until TYPE TIMESTAMPTZ USING blocked_until AT TIME ZONE 'UTC',
  ALTER COLUMN unblocked_at  TYPE TIMESTAMPTZ USING unblocked_at  AT TIME ZONE 'UTC';
