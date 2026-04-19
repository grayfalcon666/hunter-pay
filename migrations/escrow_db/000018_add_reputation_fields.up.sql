ALTER TABLE user_profiles ADD COLUMN hunter_reputation_score DOUBLE PRECISION DEFAULT 100.0;
ALTER TABLE user_profiles ADD COLUMN employer_reputation_score DOUBLE PRECISION DEFAULT 100.0;
ALTER TABLE user_profiles ADD COLUMN last_active_at TIMESTAMPTZ;
ALTER TABLE user_profiles ADD COLUMN total_good_reviews INTEGER DEFAULT 0;
ALTER TABLE user_profiles ADD COLUMN total_bad_reviews INTEGER DEFAULT 0;
ALTER TABLE user_profiles ADD COLUMN average_rating DOUBLE PRECISION DEFAULT 0.0;
ALTER TABLE user_profiles ADD COLUMN cooling_lambda DOUBLE PRECISION DEFAULT 0.05;
