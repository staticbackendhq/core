package postgresql

import "github.com/staticbackendhq/core/internal"

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
	var id string
	err = pg.DB.QueryRow(`
	INSERT INTO sb.apps(customer_id, name, allowed_domain, is_active, created)
	VALUES($1, $2, $3, $4, $5)
	RETURNING id;
	`, base.CustomerID,
		base.Name,
		base.AllowedDomain,
		base.IsActive,
		base.Created,
	).Scan(&id)
	if err != nil {
		return
	}

	b.ID = id
	return
}
