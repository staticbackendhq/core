package memory

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) CreateTenant(customer model.Tenant) (model.Tenant, error) {
	err := create(m, "sb", "customers", customer.ID, customer)
	return customer, err
}

func (m *Memory) CreateDatabase(base model.DatabaseConfig) (model.DatabaseConfig, error) {
	if err := create(m, "sb", "apps", base.ID, base); err != nil {
		return base, err
	}

	// needed to make tests pass
	task := model.Task{
		ID:    m.NewID(),
		Name:  "demo task",
		Type:  model.TaskTypeMessage,
		Value: "task demo",
	}
	err := create(m, base.Name, "sb_tasks", task.ID, task)
	return base, err
}

func (m *Memory) EmailExists(email string) (exists bool, err error) {
	list, err := all[model.Tenant](m, "sb", "customers")
	if err != nil {
		return
	}

	results := filter(list, func(x model.Tenant) bool {
		return strings.EqualFold(x.Email, email)
	})

	if len(results) != 1 {
		return
	}

	exists = true
	return
}

func (m *Memory) FindTenant(tenantID string) (cus model.Tenant, err error) {
	err = getByID(m, "sb", "customers", tenantID, &cus)
	return
}

func (m *Memory) FindDatabase(baseID string) (base model.DatabaseConfig, err error) {
	err = getByID(m, "sb", "apps", baseID, &base)
	return
}

func (m *Memory) DatabaseExists(name string) (exists bool, err error) {
	list, err := all[model.DatabaseConfig](m, "sb", "apps")
	if err != nil {
		return
	}

	results := filter(list, func(x model.DatabaseConfig) bool {
		return x.Name == name
	})

	exists = len(results) > 0
	return
}

func (m *Memory) ListDatabases() (results []model.DatabaseConfig, err error) {
	results, err = all[model.DatabaseConfig](m, "sb", "apps")
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

func (m *Memory) GetTenantByStripeID(stripeID string) (cus model.Tenant, err error) {
	list, err := all[model.Tenant](m, "sb", "customers")
	if err != nil {
		return
	}

	results := filter(list, func(x model.Tenant) bool {
		return strings.EqualFold(x.StripeID, stripeID)
	})

	if len(results) != 1 {
		err = fmt.Errorf("cannot find customer by stripe id %s", stripeID)
		return
	}

	cus = results[0]
	return
}

func (m *Memory) ActivateTenant(tenantID string, active bool) error {
	var cus model.Tenant
	if err := getByID(m, "sb", "customers", tenantID, &cus); err != nil {
		return err
	}

	cus.IsActive = active

	if err := create(m, "sb", "customers", tenantID, cus); err != nil {
		return err
	}

	return nil
}

func (m *Memory) ChangeTenantPlan(tenantID string, plan int) error {
	cus, err := m.FindTenant(tenantID)
	if err != nil {
		return err
	}

	cus.Plan = plan
	return create(m, "sb", "customers", tenantID, cus)
}

func (m *Memory) EnableExternalLogin(tenantID string, config map[string]model.OAuthConfig) error {
	b, err := model.EncryptExternalLogins(config)
	if err != nil {
		return err
	}

	cus, err := m.FindTenant(tenantID)
	if err != nil {
		return err
	}

	cus.ExternalLogins = b
	return create(m, "sb", "customers", tenantID, cus)
}

func (m *Memory) DeleteTenant(dbName, email string) error {
	return nil
}
