-- 0010_announcement_supersedes: an announcement may supersede an earlier one
-- (spec 06 Announcement.supersedes id). Additive and forward-only. The row stays
-- append-only; a superseded announcement is hidden from lists once its
-- replacement is in force, never edited or deleted.
ALTER TABLE announcements ADD COLUMN supersedes_id TEXT NOT NULL DEFAULT '';
CREATE INDEX announcements_supersedes ON announcements(network_id, supersedes_id) WHERE supersedes_id <> '';
