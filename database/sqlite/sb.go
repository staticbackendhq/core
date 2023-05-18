package sqlite

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) CreateTenant(customer model.Tenant) (c model.Tenant, err error) {
	id := sl.NewID()
	c = customer

	_, err = sl.DB.Exec(`
	INSERT INTO sb_customers(id, email, stripe_id, sub_id, plan, is_active, created)
	VALUES($1, $2, $3, $4, $5, $6, $7);
	`, id, customer.Email,
		customer.StripeID,
		customer.SubscriptionID,
		customer.Plan,
		customer.IsActive,
		customer.Created,
	)
	if err != nil {
		return
	}
	c.ID = id
	return
}

func (sl *SQLite) CreateDatabase(base model.DatabaseConfig) (b model.DatabaseConfig, err error) {
	b = base

	id := sl.NewID()
	_, err = sl.DB.Exec(`
	INSERT INTO sb_apps(id, customer_id, name, allowed_domain, is_active, monthly_email_sent, created)
	VALUES($1, $2, $3, $4, $5, $6, $7);
	`, id, base.TenantID,
		base.Name,
		strings.Join(base.AllowedDomain, "|"),
		base.IsActive,
		base.MonthlySentEmail,
		base.Created,
	)
	if err != nil {
		return
	}

	b.ID = id

	err = sl.createSystemTables(base.Name)
	return
}

func (sl *SQLite) createSystemTables(schema string) error {
	qry := strings.Replace(`
		CREATE TABLE IF NOT EXISTS {schema}_sb_accounts (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			created TIMESTAMP NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS {schema}_sb_tokens (
			id TEXT PRIMARY KEY,
			account_id TEXT REFERENCES {schema}_sb_accounts(id) ON DELETE CASCADE,
			token TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role INTEGER NOT NULL,
			reset_code TEXT NOT NULL,
			created timestamp NOT NULL			
		);

		CREATE TABLE IF NOT EXISTS {schema}_sb_forms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			data JSON NOT NULL,
			created timestamp NOT NULL
		);
		CREATE INDEX IF NOT EXISTS {schema}_sb_forms_name_idx ON {schema}_sb_forms (name);			

		CREATE TABLE IF NOT EXISTS {schema}_sb_files (
			id TEXT PRIMARY KEY,
			account_id TEXT REFERENCES {schema}_sb_accounts(id) ON DELETE CASCADE,
			key TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			size INTEGER NOT NULL,			
			uploaded timestamp NOT NULL			
		);
		CREATE INDEX IF NOT EXISTS {schema}_sb_files_acctid_idx ON {schema}_sb_files (account_id);

		CREATE TABLE IF NOT EXISTS {schema}_sb_functions (
			id TEXT PRIMARY KEY,
			function_name TEXT UNIQUE NOT NULL,
			trigger_topic TEXT NOT NULL,
			code TEXT NOT NULL,
			version INTEGER NOT NULL,
			last_updated timestamp NOT NULL,
			last_run timestamp NOT NULL
		);
		CREATE INDEX IF NOT EXISTS {schema}_sb_functions_trigger_topic_idx ON {schema}_sb_functions (trigger_topic);

		CREATE TABLE IF NOT EXISTS {schema}_sb_function_logs (
			id TEXT PRIMARY KEY,
			function_id TEXT REFERENCES {schema}_sb_functions(id) ON DELETE CASCADE,
			version INTEGER NOT NULL,
			started timestamp NOT NULL,
			completed timestamp NOT NULL,
			success BOOLEAN NOT NULL,
			output TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS {schema}_sb_tasks (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			type TEXT NOT NULL,
			value TEXT NOT NULL,
			meta TEXT NOT NULL,
			interval TEXT NOT NULL,
			last_run timestamp NOT NULL
		);
	`, "{schema}", schema, -1)

	if _, err := sl.DB.Exec(qry); err != nil {
		return err
	}

	return nil
}

func (sl *SQLite) EmailExists(email string) (bool, error) {
	var count int
	err := sl.DB.QueryRow(`
		SELECT COUNT(*) FROM sb_customers WHERE email = $1
	`, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (sl *SQLite) FindTenant(tenantID string) (customer model.Tenant, err error) {
	row := sl.DB.QueryRow(`
		SELECT * 
		FROM sb_customers
		WHERE id = $1
	`, tenantID)

	err = scanCustomer(row, &customer)
	return
}

func (sl *SQLite) FindDatabase(baseID string) (base model.DatabaseConfig, err error) {
	row := sl.DB.QueryRow(`
		SELECT * 
		FROM sb_apps 
		WHERE id = $1
	`, baseID)

	err = scanBase(row, &base)
	return
}

func (sl *SQLite) DatabaseExists(name string) (exists bool, err error) {
	var count int
	err = sl.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM sb_apps 
		WHERE name = $1
	`, name).Scan(&count)

	exists = count > 0
	return
}

func (sl *SQLite) ListDatabases() (results []model.DatabaseConfig, err error) {
	rows, err := sl.DB.Query(`
		SELECT * 
		FROM sb_apps 
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

func (sl *SQLite) IncrementMonthlyEmailSent(baseID string) error {
	_, err := sl.DB.Exec(`
		UPDATE sb_apps SET monthly_email_sent = monthly_email_sent + 1
		WHERE id = $1;
	`, baseID)

	return err
}

func (sl *SQLite) GetTenantByStripeID(stripeID string) (cus model.Tenant, err error) {
	row := sl.DB.QueryRow(`
		SELECT * 
		FROM sb_customers 
		WHERE stripe_id = $1
	`, stripeID)

	err = scanCustomer(row, &cus)
	return
}

func (sl *SQLite) ActivateTenant(tenantID string, active bool) error {
	tx, err := sl.DB.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE sb_customers SET is_active = $2 WHERE id = $1;`, tenantID, active); err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE sb_apps SET is_active = $2 WHERE customer_id = $1;`, tenantID, active); err != nil {
		return err
	}

	return tx.Commit()
}

func (sl *SQLite) ChangeTenantPlan(tenantID string, plan int) error {
	if _, err := sl.DB.Exec(`UPDATE sb_customers SET plan = $2 WHERE id = $1`, tenantID, plan); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) EnableExternalLogin(tenantID string, config map[string]model.OAuthConfig) error {
	b, err := model.EncryptExternalLogins(config)
	if err != nil {
		return err
	}

	if _, err := sl.DB.Exec(`UPDATE sb_customers SET external_logins = $2 WHERE id = $1`, tenantID, b); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) NewID() string {
	id, err := uuid.NewUUID()
	if err != nil {
		return ""
	}
	return id.String()
}

func (sl *SQLite) DeleteTenant(dbName, email string) error {
	tables, err := sl.ListCollections(dbName)
	if err != nil {
		return err
	}

	for _, table := range tables {
		if _, err := sl.DB.Exec(fmt.Sprintf("DROP TABLE %s", table)); err != nil {
			return err
		}
	}
	return nil
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
	var allowedDomain string
	err := rows.Scan(
		&b.ID,
		&b.TenantID,
		&b.Name,
		&allowedDomain,
		&b.IsActive,
		&b.MonthlySentEmail,
		&b.Created,
	)

	b.AllowedDomain = strings.Split(allowedDomain, "|")
	return err
}

func (sl *SQLite) GetAllDatabaseSizes() error {
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
