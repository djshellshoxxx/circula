-- Legacy inbox text contains a date but no exact instant. Preserve its snapshot.
ALTER TABLE notifications ADD COLUMN event_date TEXT
  CHECK (event_date IS NULL OR (body <> '' AND json_valid(event_date)));
