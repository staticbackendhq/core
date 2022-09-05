package backend

import (
	"errors"
	"strings"

	"github.com/staticbackendhq/core/internal"
)

type Tenant struct{}

func (t Tenant) CreateCustomer(cus internal.Customer) (internal.Customer, error) {
	cus.Email = strings.ToLower(cus.Email)
	if exists, err := datastore.EmailExists(cus.Email); err != nil {
		return cus, err
	} else if exists {
		return cus, errors.New("email already exists")
	}
	return datastore.CreateCustomer(cus)
}

func (t Tenant) CreateBase(base internal.BaseConfig) (internal.BaseConfig, error) {
	if exists, err := datastore.DatabaseExists(base.Name); err != nil {
		return base, err
	} else if exists {
		return base, errors.New("this database name already in use")
	}
	return datastore.CreateBase(base)
}