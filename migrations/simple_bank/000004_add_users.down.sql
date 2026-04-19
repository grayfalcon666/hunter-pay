DROP INDEX IF EXISTS "accounts_owner_currency_idx";

ALTER TABLE "accounts" DROP CONSTRAINT IF EXISTS "accounts_owner_fkey";

-- 删除 escrow 担保账户（先删除账户，再删除用户）
DELETE FROM "accounts" WHERE "owner" = 'escrow';
DELETE FROM "users" WHERE "username" = 'escrow';

DROP TABLE IF EXISTS "users";