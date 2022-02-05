-- enable the UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- the main database schema
CREATE SCHEMA IF NOT EXISTS sb;

-- needed tables for creating apps
CREATE TABLE IF NOT EXISTS sb.customers (
	id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
	email text UNIQUE NOT NULL,
	stripe_id text NOT NULL,
	sub_id text NOT NULL,
	is_active boolean NOT NULL,
	created timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS sb.apps (
	id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
	customer_id uuid REFERENCES sb.customers(id) ON DELETE CASCADE,
	name text UNIQUE NOT NULL,
	allowed_domain text[],
	is_active boolean NOT NULL,	
	monthly_email_sent integer NOT NULL,
	created timestamp NOT NULL
);