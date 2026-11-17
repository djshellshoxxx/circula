-- Rollback for 0001 (feature data is discarded; the ledger and proposals are untouched).
DROP TABLE IF EXISTS qr_idempotency;
DROP TABLE IF EXISTS qr_trade_request;
DROP TABLE IF EXISTS qr_member_alias;
