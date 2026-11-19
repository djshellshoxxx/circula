-- Persist event insertion order independently of wall-clock time and IDs.
-- Additive side tables preserve the existing append-only event records and
-- their guards. Backfill captures legacy insertion order before any VACUUM.
CREATE TABLE transaction_status_event_order (
  seq INTEGER PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE REFERENCES transaction_status_events(id)
);
INSERT INTO transaction_status_event_order(seq,event_id)
  SELECT rowid,id FROM transaction_status_events ORDER BY rowid;
-- REPLACE performs an implicit delete which does not fire delete triggers
-- unless recursive_triggers is enabled. Reject key reuse before insertion.
CREATE TRIGGER status_event_order_no_replace BEFORE INSERT ON transaction_status_event_order
WHEN EXISTS (SELECT 1 FROM transaction_status_event_order WHERE seq = NEW.seq OR event_id = NEW.event_id)
BEGIN SELECT RAISE(ABORT, 'event order is append-only'); END;
CREATE TRIGGER status_event_record_order AFTER INSERT ON transaction_status_events
BEGIN INSERT INTO transaction_status_event_order(event_id) VALUES (NEW.id); END;
CREATE TRIGGER status_event_order_no_update BEFORE UPDATE ON transaction_status_event_order
BEGIN SELECT RAISE(ABORT, 'event order is append-only'); END;
CREATE TRIGGER status_event_order_no_delete BEFORE DELETE ON transaction_status_event_order
BEGIN SELECT RAISE(ABORT, 'event order is append-only'); END;

CREATE TABLE dispute_event_order (
  seq INTEGER PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE REFERENCES dispute_events(id)
);
INSERT INTO dispute_event_order(seq,event_id)
  SELECT rowid,id FROM dispute_events ORDER BY rowid;
CREATE TRIGGER dispute_event_order_no_replace BEFORE INSERT ON dispute_event_order
WHEN EXISTS (SELECT 1 FROM dispute_event_order WHERE seq = NEW.seq OR event_id = NEW.event_id)
BEGIN SELECT RAISE(ABORT, 'event order is append-only'); END;
CREATE TRIGGER dispute_event_record_order AFTER INSERT ON dispute_events
BEGIN INSERT INTO dispute_event_order(event_id) VALUES (NEW.id); END;
CREATE TRIGGER dispute_event_order_no_update BEFORE UPDATE ON dispute_event_order
BEGIN SELECT RAISE(ABORT, 'event order is append-only'); END;
CREATE TRIGGER dispute_event_order_no_delete BEFORE DELETE ON dispute_event_order
BEGIN SELECT RAISE(ABORT, 'event order is append-only'); END;
