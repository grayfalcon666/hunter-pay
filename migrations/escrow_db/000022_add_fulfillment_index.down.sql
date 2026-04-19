ALTER TABLE user_profiles
    DROP COLUMN IF EXISTS fulfillment_index,
    DROP COLUMN IF EXISTS cooling_lambda,
    DROP COLUMN IF EXISTS task_window_size,
    DROP COLUMN IF EXISTS version;
