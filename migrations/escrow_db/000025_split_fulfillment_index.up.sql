ALTER TABLE user_profiles
    DROP COLUMN IF EXISTS fulfillment_index,
    ADD COLUMN IF NOT EXISTS hunter_fulfillment_index INTEGER DEFAULT 50,
    ADD COLUMN IF NOT EXISTS employer_fulfillment_index INTEGER DEFAULT 50;
