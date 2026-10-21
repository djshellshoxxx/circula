-- 0008_balance_bound: A3-03 hard bound on cached account balances. Forward-only.
--
-- The service already refuses a posting that would push either side of an
-- entry pair outside +-2^62 (the system account too). These triggers are the
-- database-level backstop: no code path can store a balance outside the range,
-- so SUM() over a network can no longer overflow int64 in VerifyLedger.
--
-- Triggers rather than a CHECK: SQLite cannot add a CHECK to an existing table
-- without rebuilding accounts (and every table that references it). Triggers
-- fire only on later writes, so existing rows are untouched and an upgrade
-- cannot fail on legacy data.

CREATE TRIGGER accounts_balance_bound_insert BEFORE INSERT ON accounts
WHEN NEW.balance > 4611686018427387904 OR NEW.balance < -4611686018427387904
BEGIN SELECT RAISE(ABORT, 'account balance out of range'); END;

CREATE TRIGGER accounts_balance_bound_update BEFORE UPDATE OF balance ON accounts
WHEN NEW.balance > 4611686018427387904 OR NEW.balance < -4611686018427387904
BEGIN SELECT RAISE(ABORT, 'account balance out of range'); END;
