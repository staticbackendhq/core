-- needed tables for creating apps
CREATE TABLE IF NOT EXISTS sb_customers (
	id TEXT PRIMARY KEY,
	email TEXT UNIQUE NOT NULL,
	stripe_id TEXT NOT NULL,
	sub_id TEXT NOT NULL,
	plan INTEGER NOT NULL DEFAULT 0,
	external_logins BLOB,
	is_active BOOLEAN NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sb_apps (
	id TEXT PRIMARY KEY,
	customer_id TEXT REFERENCES sb_customers(id) ON DELETE CASCADE,
	name TEXT UNIQUE NOT NULL,
	allowed_domain TEXT,
	is_active BOOLEAN NOT NULL,	
	monthly_email_sent INTEGER NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);