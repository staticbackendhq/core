package postgresql

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) CreateTenant(customer model.Tenant) (c model.Tenant, err error) {
	var id string
	c = customer

	err = pg.DB.QueryRow(`
	INSERT INTO sb.customers(email, stripe_id, sub_id, plan, is_active, created)
	VALUES($1, $2, $3, $4, $5, $6)
	RETURNING id;
	`, customer.Email,
		customer.StripeID,
		customer.SubscriptionID,
		customer.Plan,
		customer.IsActive,
		customer.Created,
	).Scan(&id)
	if err != nil {
		return
	}
	c.ID = id
	return
}

func (pg *PostgreSQL) CreateDatabase(base model.DatabaseConfig) (b model.DatabaseConfig, err error) {
	b = base

	_, err = pg.DB.Exec(fmt.Sprintf("CREATE SCHEMA %s;", b.Name))
	if err != nil {
		return
	}

	var id string
	err = pg.DB.QueryRow(`
	INSERT INTO sb.apps(customer_id, name, allowed_domain, is_active, monthly_email_sent, created)
	VALUES($1, $2, $3, $4, $5, $6)
	RETURNING id;
	`, base.TenantID,
		base.Name,
		pq.Array(base.AllowedDomain),
		base.IsActive,
		base.MonthlySentEmail,
		base.Created,
	).Scan(&id)
	if err != nil {
		return
	}

	b.ID = id

	err = pg.createSystemTables(base.Name)
	return
}

func (pg *PostgreSQL) createSystemTables(schema string) error {
	qry := strings.Replace(`
		CREATE TABLE IF NOT EXISTS {schema}.sb_accounts (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			email TEXT UNIQUE NOT NULL,
			created timestamp NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS {schema}.sb_tokens (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			account_id uuid REFERENCES {schema}.sb_accounts(id) ON DELETE CASCADE,
			token TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role INTEGER NOT NULL,
			reset_code TEXT NOT NULL,
			created timestamp NOT NULL			
		);

		CREATE TABLE IF NOT EXISTS {schema}.sb_forms (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			name TEXT NOT NULL,
			data JSONB NOT NULL,
			created timestamp NOT NULL
		);
		CREATE INDEX IF NOT EXISTS sb_forms_name_idx ON {schema}.sb_forms (name);			

		CREATE TABLE IF NOT EXISTS {schema}.sb_files (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			account_id uuid REFERENCES {schema}.sb_accounts(id) ON DELETE CASCADE,
			key TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			size INTEGER NOT NULL,			
			uploaded timestamp NOT NULL			
		);
		CREATE INDEX IF NOT EXISTS sb_files_acctid_idx ON {schema}.sb_files (account_id);

		CREATE TABLE IF NOT EXISTS {schema}.sb_functions (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			function_name TEXT UNIQUE NOT NULL,
			trigger_topic TEXT NOT NULL,
			code TEXT NOT NULL,
			version INTEGER NOT NULL,
			last_updated timestamp NOT NULL,
			last_run timestamp NOT NULL
		);
		CREATE INDEX IF NOT EXISTS sb_functions_trigger_topic_idx ON {schema}.sb_functions (trigger_topic);

		CREATE TABLE IF NOT EXISTS {schema}.sb_function_logs (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			function_id uuid REFERENCES {schema}.sb_functions(id) ON DELETE CASCADE,
			version INTEGER NOT NULL,
			started timestamp NOT NULL,
			completed timestamp NOT NULL,
			success BOOLEAN NOT NULL,
			output TEXT[] NOT NULL
		);

		CREATE TABLE IF NOT EXISTS {schema}.sb_tasks (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			name TEXT UNIQUE NOT NULL,
			type TEXT NOT NULL,
			value TEXT NOT NULL,
			meta TEXT NOT NULL,
			interval TEXT NOT NULL,
			last_run timestamp NOT NULL
		);
	`, "{schema}", schema, -1)

	if _, err := pg.DB.Exec(qry); err != nil {
		return err
	}

	return nil
}

func (pg *PostgreSQL) EmailExists(email string) (bool, error) {
	var count int
	err := pg.DB.QueryRow(`
		SELECT COUNT(*) FROM sb.customers WHERE email = $1
	`, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (pg *PostgreSQL) FindTenant(tenantID string) (customer model.Tenant, err error) {
	row := pg.DB.QueryRow(`
		SELECT * 
		FROM sb.customers
		WHERE id = $1
	`, tenantID)

	err = scanCustomer(row, &customer)
	return
}

func (pg *PostgreSQL) FindDatabase(baseID string) (base model.DatabaseConfig, err error) {
	row := pg.DB.QueryRow(`
		SELECT * 
		FROM sb.apps 
		WHERE id = $1
	`, baseID)

	err = scanBase(row, &base)
	return
}

func (pg *PostgreSQL) DatabaseExists(name string) (exists bool, err error) {
	var count int
	err = pg.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM sb.apps 
		WHERE name = $1
	`, name).Scan(&count)

	exists = count > 0
	return
}

func (pg *PostgreSQL) ListDatabases() (results []model.DatabaseConfig, err error) {
	rows, err := pg.DB.Query(`
		SELECT * 
		FROM sb.apps 
		WHERE is_active = true
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var base model.DatabaseConfig
		if err = scanBase(rows, &base); err != nil {
			return
		}

		results = append(results, base)
	}

	err = rows.Err()
	return
}

func (pg *PostgreSQL) IncrementMonthlyEmailSent(baseID string) error {
	_, err := pg.DB.Exec(`
		UPDATE sb.apps SET monthly_email_sent = monthly_email_sent + 1
		WHERE id = $1;
	`, baseID)

	return err
}

func (pg *PostgreSQL) GetTenantByStripeID(stripeID string) (cus model.Tenant, err error) {
	row := pg.DB.QueryRow(`
		SELECT * 
		FROM sb.customers 
		WHERE stripe_id = $1
	`, stripeID)

	err = scanCustomer(row, &cus)
	return
}

func (pg *PostgreSQL) ActivateTenant(tenantID string, active bool) error {
	tx, err := pg.DB.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE sb.customers SET is_active = $2 WHERE id = $1;`, tenantID, active); err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE sb.apps SET is_active = $2 WHERE customer_id = $1;`, tenantID, active); err != nil {
		return err
	}

	return tx.Commit()
}

func (pg *PostgreSQL) ChangeTenantPlan(tenantID string, plan int) error {
	if _, err := pg.DB.Exec(`UPDATE sb.customers SET plan = $2 WHERE id = $1`, tenantID, plan); err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) EnableExternalLogin(tenantID string, config map[string]model.OAuthConfig) error {
	b, err := model.EncryptExternalLogins(config)
	if err != nil {
		return err
	}

	if _, err := pg.DB.Exec(`UPDATE sb.customers SET external_logins = $2 WHERE id = $1`, tenantID, b); err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) NewID() string {
	var id string
	if err := pg.DB.QueryRow(`SELECT uuid_generate_v4 ()`).Scan(&id); err != nil {
		pg.log.Error().Err(err).Msg("error in postgresql.NewID")
		return ""
	}
	return id
}

func (pg *PostgreSQL) DeleteTenant(dbName, email string) error {
	_, err := pg.DB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE;`, dbName))
	if err != nil {
		return err
	}

	_, err = pg.DB.Exec(`
		DELETE FROM sb.customers WHERE email = $1;
	`, email)

	return err
}

func scanCustomer(rows Scanner, c *model.Tenant) error {
	return rows.Scan(
		&c.ID,
		&c.Email,
		&c.StripeID,
		&c.SubscriptionID,
		&c.IsActive,
		&c.Created,
		&c.Plan,
		&c.ExternalLogins,
	)
}

func scanBase(rows Scanner, b *model.DatabaseConfig) error {
	return rows.Scan(
		&b.ID,
		&b.TenantID,
		&b.Name,
		pq.Array(&b.AllowedDomain),
		&b.IsActive,
		&b.MonthlySentEmail,
		&b.Created,
	)
}

func (pg *PostgreSQL) GetAllDatabaseSizes() error {
	/*qry := `
		SELECT
			schema_name,
			pg_size_pretty(sum(table_size)::bigint),
		FROM (
			SELECT pg_catalog.pg_namespace.nspname as schema_name,
				pg_relation_size(pg_catalog.pg_class.oid) as table_size
			FROM pg_catalog.pg_class
			JOIN pg_catalog.pg_namespace ON relnamespace = pg_catalog.pg_namespace.oid
		) t
		GROUP BY schema_name
		ORDER BY schema_name
	`*/
	return nil
}
