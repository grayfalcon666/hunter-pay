-- No rollback needed for backfill; manually delete from all_messages if needed
DELETE FROM all_messages WHERE message_type = 'bounty';
