-- Create experience_level enum type
DO $$ BEGIN
    CREATE TYPE experience_level AS ENUM ('ENTRY', 'JUNIOR', 'MID', 'SENIOR', 'EXPERT');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create review_type enum type
DO $$ BEGIN
    CREATE TYPE review_type AS ENUM ('EMPLOYER_TO_HUNTER', 'HUNTER_TO_EMPLOYER');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS user_profiles (
    username                  VARCHAR(255) PRIMARY KEY,
    expected_salary_min        VARCHAR(50) DEFAULT '0',
    expected_salary_max        VARCHAR(50) DEFAULT '0',
    work_location              VARCHAR(255) DEFAULT '',
    experience_level           experience_level DEFAULT 'ENTRY',
    bio                        TEXT DEFAULT '',
    avatar_url                 VARCHAR(500) DEFAULT '',
    completion_rate            DOUBLE PRECISION DEFAULT 0.0,
    good_review_rate           DOUBLE PRECISION DEFAULT 0.0,
    total_bounties_posted      INT DEFAULT 0,
    total_bounties_completed   INT DEFAULT 0,
    total_earnings             BIGINT DEFAULT 0,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_reviews (
    id                  BIGSERIAL PRIMARY KEY,
    reviewer_username   VARCHAR(255) NOT NULL,
    reviewed_username   VARCHAR(255) NOT NULL,
    bounty_id           BIGINT NOT NULL,
    rating              INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment             TEXT DEFAULT '',
    review_type         review_type NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reviews_reviewed ON user_reviews(reviewed_username);
CREATE INDEX IF NOT EXISTS idx_reviews_bounty ON user_reviews(bounty_id);
