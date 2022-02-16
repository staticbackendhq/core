package postgresql

import (
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
	"github.com/staticbackendhq/core/internal"
)

func (pg *PostgreSQL) CreateCustomer(customer internal.Customer) (c internal.Customer, err error) {
	var id string
	c = customer

	err = pg.DB.QueryRow(`
	INSERT INTO sb.customers(email, stripe_id, sub_id, is_active, created)
	VALUES($1, $2, $3, $4, $5)
	RETURNING id;
	`, customer.Email,
		customer.StripeID,
		customer.SubscriptionID,
		customer.IsActive,
		customer.Created,
	).Scan(&id)
	if err != nil {
		return
	}
	c.ID = id
	return
}

func (pg *PostgreSQL) CreateBase(base internal.BaseConfig) (b internal.BaseConfig, err error) {
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
	`, base.CustomerID,
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

		CREATE TABLE IF NOT EXISTS {schema}.sb_files (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			account_id uuid REFERENCES {schema}.sb_accounts(id) ON DELETE CASCADE,
			key TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			size INTEGER NOT NULL,			
			uploaded timestamp NOT NULL			
		);

		CREATE TABLE IF NOT EXISTS {schema}.sb_functions (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			function_name TEXT UNIQUE NOT NULL,
			trigger_topic TEXT NOT NULL,
			code TEXT NOT NULL,
			version INTEGER NOT NULL,
			last_updated timestamp NOT NULL,
			last_run timestamp NOT NULL
		);

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

func (pg *PostgreSQL) FindAccount(customerID string) (customer internal.Customer, err error) {
	row := pg.DB.QueryRow(`
		SELECT * 
		FROM sb.customers
		WHERE id = $1
	`, customerID)

	err = scanCustomer(row, &customer)
	return
}

func (pg *PostgreSQL) FindDatabase(baseID string) (base internal.BaseConfig, err error) {
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

func (pg *PostgreSQL) ListDatabases() (results []internal.BaseConfig, err error) {
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
		var base internal.BaseConfig
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

func (pg *PostgreSQL) GetCustomerByStripeID(stripeID string) (cus internal.Customer, err error) {
	row := pg.DB.QueryRow(`
		SELECT * 
		FROM sb.customers 
		WHERE stripe_id = $1
	`, stripeID)

	err = scanCustomer(row, &cus)
	return
}

func (pg *PostgreSQL) ActivateCustomer(customerID string) error {
	tx, err := pg.DB.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE sb.customers SET is_active = true WHERE id = $1;`, customerID); err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE sb.apps SET is_active = true WHERE customer_id = $1;`, customerID); err != nil {
		return err
	}

	return tx.Commit()
}

func (pg *PostgreSQL) NewID() string {
	var id string
	if err := pg.DB.QueryRow(`SELECT uuid_generate_v4 ()`).Scan(&id); err != nil {
		//TODO: do something with this error
		log.Println("error in postgresql.NewID: ", err)
		return ""
	}
	return id
}

func (pg *PostgreSQL) DeleteCustomer(dbName, email string) error {
	_, err := pg.DB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE;`, dbName))
	if err != nil {
		return err
	}

	_, err = pg.DB.Exec(`
		DELETE FROM sb.customers WHERE email = $1;
	`, email)

	return err
}

func scanCustomer(rows Scanner, c *internal.Customer) error {
	return rows.Scan(
		&c.ID,
		&c.Email,
		&c.StripeID,
		&c.SubscriptionID,
		&c.IsActive,
		&c.Created,
	)
}

func scanBase(rows Scanner, b *internal.BaseConfig) error {
	return rows.Scan(
		&b.ID,
		&b.CustomerID,
		&b.Name,
		pq.Array(&b.AllowedDomain),
		&b.IsActive,
		&b.MonthlySentEmail,
		&b.Created,
	)
}
