CREATE TABLE IF NOT EXISTS sb.migrations (
	id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
	version INTEGER NOT NULL,
	files TEXT NOT NULL,
	executed timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO sb.migrations(version, files)
VALUES(2, '0002_add_migrations_table.sql');