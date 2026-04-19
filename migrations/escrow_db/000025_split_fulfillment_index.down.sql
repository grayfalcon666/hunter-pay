ALTER TABLE user_profiles
    DROP COLUMN IF EXISTS hunter_fulfillment_index,
    DROP COLUMN IF EXISTS employer_fulfillment_index,
    ADD COLUMN IF NOT EXISTS fulfillment_index INTEGER DEFAULT 50;
