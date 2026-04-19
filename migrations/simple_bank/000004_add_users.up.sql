CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "accounts" ADD FOREIGN KEY ("owner") REFERENCES "users" ("username");

CREATE UNIQUE INDEX ON "accounts" ("owner", "currency");

-- 插入 escrow 担保账户（username="escrow"）
-- 这是平台默认的担保账户，用于托管 bounty 资金
INSERT INTO "users" ("username", "hashed_password", "full_name", "email", "password_changed_at", "created_at")
VALUES ('escrow', '', 'Escrow Platform Account', 'escrow@platform.local', '0001-01-01 00:00:00Z', '0001-01-01 00:00:00Z')
ON CONFLICT ("username") DO NOTHING;

-- 为 escrow 账户创建 USD 账户（用于接收和托管 bounty 资金）
INSERT INTO "accounts" ("owner", "balance", "currency", "created_at")
VALUES ('escrow', 0, 'USD', NOW())
ON CONFLICT DO NOTHING;