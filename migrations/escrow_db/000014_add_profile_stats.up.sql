ALTER TABLE user_profiles ADD COLUMN reputation_score DOUBLE PRECISION DEFAULT 100.0;
ALTER TABLE user_profiles ADD COLUMN last_completed_at TIMESTAMPTZ;
