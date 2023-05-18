CREATE TABLE IF NOT EXISTS sb_migrations (
	id TEXT PRIMARY KEY,
	version INTEGER NOT NULL,
	files TEXT NOT NULL,
	executed TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO sb_migrations(id, version, files)
VALUES('0002', 2, '0002_add_migrations_table.sql');