CREATE TABLE IF NOT EXISTS message_threads (
 network TEXT NOT NULL, id TEXT NOT NULL, a TEXT NOT NULL, b TEXT NOT NULL,
 reference TEXT NOT NULL DEFAULT '', created INTEGER NOT NULL, activity INTEGER NOT NULL,
 retention INTEGER NOT NULL CHECK(retention BETWEEN 1 AND 30), sequence INTEGER NOT NULL DEFAULT 0,
 closed INTEGER NOT NULL DEFAULT 0, held INTEGER NOT NULL DEFAULT 0, purged INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY(network,id), CHECK(a <> b)
);
CREATE INDEX IF NOT EXISTS message_threads_members ON message_threads(network,a,b,activity);
CREATE TABLE IF NOT EXISTS message_entries (
 network TEXT NOT NULL, thread TEXT NOT NULL, sequence INTEGER NOT NULL,
 id TEXT NOT NULL, sender TEXT NOT NULL, body TEXT NOT NULL, created INTEGER NOT NULL,
 PRIMARY KEY(network,thread,sequence), UNIQUE(network,id),
 FOREIGN KEY(network,thread) REFERENCES message_threads(network,id)
);
CREATE INDEX IF NOT EXISTS message_send_quota ON message_entries(network,sender,created);
CREATE TABLE IF NOT EXISTS message_positions (
 network TEXT NOT NULL, thread TEXT NOT NULL, member TEXT NOT NULL,
 seen INTEGER NOT NULL DEFAULT 0, hidden INTEGER NOT NULL DEFAULT 0, hidden_view INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY(network,thread,member), FOREIGN KEY(network,thread) REFERENCES message_threads(network,id)
);
CREATE TABLE IF NOT EXISTS message_blocks (
 network TEXT NOT NULL, actor TEXT NOT NULL, target TEXT NOT NULL,
 PRIMARY KEY(network,actor,target), CHECK(actor <> target)
);
CREATE TABLE IF NOT EXISTS message_idempotency (
 network TEXT NOT NULL, actor TEXT NOT NULL, operation TEXT NOT NULL, key TEXT NOT NULL,
 hash TEXT NOT NULL, result TEXT NOT NULL, thread TEXT NOT NULL,
 PRIMARY KEY(network,actor,operation,key), FOREIGN KEY(network,thread) REFERENCES message_threads(network,id)
);
CREATE TABLE IF NOT EXISTS message_reports (
 network TEXT NOT NULL, thread TEXT NOT NULL, sequence INTEGER NOT NULL, reporter TEXT NOT NULL,
 reason TEXT NOT NULL, created INTEGER NOT NULL, state TEXT NOT NULL DEFAULT 'open',
 PRIMARY KEY(network,thread,sequence,reporter),
 FOREIGN KEY(network,thread,sequence) REFERENCES message_entries(network,thread,sequence)
);

CREATE INDEX IF NOT EXISTS message_creation_quota ON message_threads(network,a,created);
