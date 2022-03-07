package memory

import (
	"strings"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) CreateCustomer(customer internal.Customer) (internal.Customer, error) {
	customer.ID = m.NewID()
	err := m.create("sb", "customers", customer.ID, customer)
	return customer, err
}

func (m *Memory) CreateBase(base internal.BaseConfig) (internal.BaseConfig, error) {
	base.ID = m.NewID()
	err := m.create("sb", "apps", base.ID, base)
	return base, err
}

func (m *Memory) EmailExists(email string) (exists bool, err error) {
	list, err := m.all("sb", "customers")
	if err != nil {
		return
	}

	for _, cus := range list {
		s, ok := cus["email"].(string)
		if !ok {
			continue
		}

		if strings.EqualFold(s, email) {
			exists = true
			break
		}
	}
	return
}

func (m *Memory) FindAccount(customerID string) (cus internal.Customer, err error) {
	err = m.getByID("sb", "customers", customerID, &cus)
	return
}

func (m *Memory) FindDatabase(baseID string) (base internal.BaseConfig, err error) {
	err = m.getByID("sb", "apps", baseID, &base)
	return
}

func (m *Memory) DatabaseExists(name string) (exists bool, err error) {
	list, err := m.all("sb", "apps")
	if err != nil {
		return
	}

	results, err := m.filter(list, FilterParam{"name", "=", name})
	if err != nil {
		return
	}

	exists = len(results) > 0
	return
}

func (m *Memory) ListDatabases() (results []internal.BaseConfig, error) {
	list, err := m.all("sb", "apps")
	if err != nil {
		return
	}

	for _, item := range list {
		
	}
}

}