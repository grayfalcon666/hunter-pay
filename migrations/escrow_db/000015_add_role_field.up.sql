ALTER TABLE user_profiles ADD COLUMN role VARCHAR(20) DEFAULT 'POSTER' NOT NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_role ON user_profiles(role);
