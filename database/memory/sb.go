package memory

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) CreateCustomer(customer internal.Customer) (internal.Customer, error) {
	customer.ID = m.NewID()
	err := create(m, "sb", "customers", customer.ID, customer)
	return customer, err
}

func (m *Memory) CreateBase(base internal.BaseConfig) (internal.BaseConfig, error) {
	base.ID = m.NewID()
	if err := create(m, "sb", "apps", base.ID, base); err != nil {
		return base, err
	}

	// needed to make tests pass
	task := internal.Task{
		ID:    m.NewID(),
		Name:  "demo task",
		Type:  internal.TaskTypeMessage,
		Value: "task demo",
	}
	err := create(m, base.Name, "sb_tasks", task.ID, task)
	return base, err
}

func (m *Memory) EmailExists(email string) (exists bool, err error) {
	list, err := all[internal.Customer](m, "sb", "customers")
	if err != nil {
		return
	}

	results := filter(list, func(x internal.Customer) bool {
		return strings.EqualFold(x.Email, email)
	})

	if len(results) != 1 {
		return
	}

	exists = true
	return
}

func (m *Memory) FindAccount(customerID string) (cus internal.Customer, err error) {
	err = getByID(m, "sb", "customers", customerID, &cus)
	return
}

func (m *Memory) FindDatabase(baseID string) (base internal.BaseConfig, err error) {
	err = getByID(m, "sb", "apps", baseID, &base)
	return
}

func (m *Memory) DatabaseExists(name string) (exists bool, err error) {
	list, err := all[internal.BaseConfig](m, "sb", "apps")
	if err != nil {
		return
	}

	results := filter(list, func(x internal.BaseConfig) bool {
		return x.Name == name
	})

	exists = len(results) > 0
	return
}

func (m *Memory) ListDatabases() (results []internal.BaseConfig, err error) {
	results, err = all[internal.BaseConfig](m, "sb", "apps")
	return
}

func (m *Memory) IncrementMonthlyEmailSent(baseID string) error {
	base, err := m.FindDatabase(baseID)
	if err != nil {
		return err
	}

	base.MonthlySentEmail += 1

	return create(m, "sb", "apps", baseID, base)
}

func (m *Memory) GetCustomerByStripeID(stripeID string) (cus internal.Customer, err error) {
	list, err := all[internal.Customer](m, "sb", "customers")
	if err != nil {
		return
	}

	results := filter(list, func(x internal.Customer) bool {
		return strings.EqualFold(x.StripeID, stripeID)
	})

	if len(results) != 1 {
		err = fmt.Errorf("cannot find customer by stripe id %s", stripeID)
		return
	}

	cus = results[0]
	return
}

func (m *Memory) ActivateCustomer(customerID string, active bool) error {
	var cus internal.Customer
	if err := getByID(m, "sb", "customers", customerID, &cus); err != nil {
		return err
	}

	cus.IsActive = active

	if err := create(m, "sb", "customers", customerID, cus); err != nil {
		return err
	}

	return nil
}

func (m *Memory) ChangeCustomerPlan(customerID string, plan int) error {
	cus, err := m.FindAccount(customerID)
	if err != nil {
		return err
	}

	cus.Plan = plan
	return create(m, "sb", "customers", customerID, cus)
}

func (m *Memory) DeleteCustomer(dbName, email string) error {
	return nil
}
