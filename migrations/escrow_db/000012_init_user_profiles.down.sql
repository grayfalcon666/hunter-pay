DROP INDEX IF EXISTS idx_reviews_bounty;
DROP INDEX IF EXISTS idx_reviews_reviewed;
DROP TABLE IF EXISTS user_reviews CASCADE;
DROP TABLE IF EXISTS user_profiles CASCADE;
DROP TYPE IF EXISTS review_type CASCADE;
DROP TYPE IF EXISTS experience_level CASCADE;
