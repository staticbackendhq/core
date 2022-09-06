package database

import (
	"github.com/staticbackendhq/core/model"
)

const (
	DataStorePostgreSQL = "postgresql"
	DataStoreMongoDB    = "mongo"
	DataStoreMemory     = "memory"
)

// Persister used for anything that persists to the database
type Persister interface {
	// Ping sends a ping to the db engine
	Ping() error
	// CreateIndex creates database index for a specific field in a collection
	CreateIndex(dbName, col, field string) error

	// tenant / database related
	// CreateTenant creates a tenant
	CreateTenant(model.Tenant) (model.Tenant, error)
	// CreateDatabase creates a database for a tenant
	CreateDatabase(model.DatabaseConfig) (model.DatabaseConfig, error)
	// EmailExists checks if this tenant email exists
	EmailExists(email string) (bool, error)
	// FindTenant returns a tenant by its ID
	FindTenant(tenantID string) (model.Tenant, error)
	// FindDatabase returns a database matching by its ID
	FindDatabase(baseID string) (model.DatabaseConfig, error)
	// DatabaseExists checks if this database name exists
	DatabaseExists(name string) (bool, error)
	// ListDatabases lists all databases in this system
	ListDatabases() ([]model.DatabaseConfig, error)
	// IncrementMonthlyEmailSent increments the monthly email sending counter
	IncrementMonthlyEmailSent(baseID string) error
	// GetTenantByStripeID finds a tenant by its Stripe customer ID
	GetTenantByStripeID(stripeID string) (cus model.Tenant, err error)
	// ActivateTenant turns the IsActive flag for the tenant and its database
	ActivateTenant(tenantID string, active bool) error
	// ChangeTenantPlan updates the subscription plan
	ChangeTenantPlan(tenantID string, plan int) error
	// EnableExternalLogin adds or creates a new config for an external login provider
	EnableExternalLogin(tenantID string, config map[string]model.OAuthConfig) error
	// NewID generates a unique identifier that can be used in your model
	NewID() string
	// DeleteTenant removes the database and tenant
	// note: this does not remove all the tenant's data
	DeleteTenant(dbName, email string) error

	// system user account functions
	// FindUser find a user by its ID
	FindUser(dbName, userID, token string) (model.User, error)
	// FindRootUser validates that those credentials are the root user for a database
	FindRootUser(dbName, userID, accountID, token string) (model.User, error)
	// GetRootForBase returns the root user for a database
	GetRootForBase(dbName string) (model.User, error)
	// FindUserByEmail returns the user by its email
	FindUserByEmail(dbName, email string) (model.User, error)
	// UserEmailExists checks if a user email exists in a database
	UserEmailExists(dbName, email string) (exists bool, err error)
	// GetFirstUserFromAccountID get the first user created for an account
	GetFirstUserFromAccountID(dbName, accountID string) (model.User, error)

	// membership / account & user functions
	// CreateAccount creates an account
	CreateAccount(dbName, email string) (id string, err error)
	// CreateUser creates a user for an account
	CreateUser(dbName string, tok model.User) (id string, err error)
	// SetPasswordResetCode sets the forge password code
	SetPasswordResetCode(dbName, tokenID, code string) error
	// ResetPassword resets a user password
	ResetPassword(dbName, email, code, password string) error
	// SetUserRole sets a user's role
	SetUserRole(dbName, email string, role int) error
	// UserSetPassword user initiated password reset
	UserSetPassword(dbName, userID, password string) error

	// base CRUD
	// CreateDocument creates a record in a collection
	CreateDocument(auth model.Auth, dbName, col string, doc map[string]interface{}) (map[string]interface{}, error)
	// BulkCreateDocument creates records in bulk in a collection
	BulkCreateDocument(auth model.Auth, dbName, col string, docs []interface{}) error
	// ListDocuments lists records from a collection ordered/sorted by params
	ListDocuments(auth model.Auth, dbName, col string, params model.ListParams) (model.PagedResult, error)
	// QueryDocuments filters record based on criterias ordered/sorted by params
	QueryDocuments(auth model.Auth, dbName, col string, filter map[string]interface{}, params model.ListParams) (model.PagedResult, error)
	// GetDocumentByID returns a record by its ID
	GetDocumentByID(auth model.Auth, dbName, col, id string) (map[string]interface{}, error)
	// GetDocumentsByIDs returns a list of records by multiple ids
	GetDocumentsByIDs(auth model.Auth, dbName, col string, ids []string) ([]map[string]interface{}, error)
	// UpdateDocument updates a full or partial record
	UpdateDocument(auth model.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error)
	// UpdateDocuments updates multiple records matching filters
	UpdateDocuments(auth model.Auth, dbName, col string, filters map[string]interface{}, updateFields map[string]interface{}) (int64, error)
	// IncrementValue increments/decrements a specific field in a record
	IncrementValue(auth model.Auth, dbName, col, id, field string, n int) error
	// DeleteDocument removes a record by its ID
	DeleteDocument(auth model.Auth, dbName, col, id string) (int64, error)
	// ListCollections returns all collections for a database
	ListCollections(dbName string) ([]string, error)
	// ParseQuery parses the filters into an internal query clauses
	ParseQuery(clauses [][]interface{}) (map[string]interface{}, error)

	// form functions
	// AddFormSubmission adds a form submission
	AddFormSubmission(dbName, form string, doc map[string]interface{}) error
	// ListFormSubmissions lists all submissions for a form
	ListFormSubmissions(dbName, name string) ([]map[string]interface{}, error)
	// GetForms returns all forms
	GetForms(dbName string) ([]string, error)

	// Function functions
	// AddFunction creates a server-side function
	AddFunction(dbName string, data model.ExecData) (string, error)
	// UpdateFunction updates a server-side function
	UpdateFunction(dbName, id, code, trigger string) error
	// GetFunctionForExecution returns a function ready for execution
	GetFunctionForExecution(dbName, name string) (model.ExecData, error)
	// GetFunctionByID returns a function by its ID
	GetFunctionByID(dbName, id string) (model.ExecData, error)
	// GetFunctionByNamereturns a function by its name
	GetFunctionByName(dbName, name string) (model.ExecData, error)
	// ListFunctions lists all functions
	ListFunctions(dbName string) ([]model.ExecData, error)
	// ListFunctionsByTrigger lists all functions for a specific trigger
	ListFunctionsByTrigger(dbName, trigger string) ([]model.ExecData, error)
	// DeleteFunction removes a function
	DeleteFunction(dbName, name string) error
	// RanFunction records a function execution and its output
	RanFunction(dbName, id string, rh model.ExecHistory) error

	// schedule tasks
	ListTasks() ([]model.Task, error)

	// Files / storage
	// AddFile adds a new file
	AddFile(dbName string, f model.File) (id string, err error)
	// GetFileByID get a file by its ID
	GetFileByID(dbName, fileID string) (f model.File, err error)
	// DeleteFile removes a file
	DeleteFile(dbName, fileID string) error
	// ListAllFiles lists all file
	ListAllFiles(dbName, accountID string) ([]model.File, error)
}
