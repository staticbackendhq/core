package database

import (
	"github.com/staticbackendhq/core/model"
)

const (
	DataStorePostgreSQL = "postgresql"
	DataStoreMongoDB    = "mongo"
	DataStoreMemory     = "memory"
)

type Persister interface {
	Ping() error
	CreateIndex(dbName, col, field string) error

	// customer / app related
	CreateCustomer(model.Customer) (model.Customer, error)
	CreateBase(model.BaseConfig) (model.BaseConfig, error)
	EmailExists(email string) (bool, error)
	FindAccount(customerID string) (model.Customer, error)
	FindDatabase(baseID string) (model.BaseConfig, error)
	DatabaseExists(name string) (bool, error)
	ListDatabases() ([]model.BaseConfig, error)
	IncrementMonthlyEmailSent(baseID string) error
	GetCustomerByStripeID(stripeID string) (cus model.Customer, err error)
	ActivateCustomer(customerID string, active bool) error
	ChangeCustomerPlan(customerID string, plan int) error
	EnableExternalLogin(customerID string, config map[string]model.OAuthConfig) error
	NewID() string
	DeleteCustomer(dbName, email string) error

	// system user account function s
	FindToken(dbName, tokenID, token string) (model.Token, error)
	FindRootToken(dbName, tokenID, accountID, token string) (model.Token, error)
	GetRootForBase(dbName string) (model.Token, error)
	FindTokenByEmail(dbName, email string) (model.Token, error)
	UserEmailExists(dbName, email string) (exists bool, err error)
	GetFirstTokenFromAccountID(dbName, accountID string) (tok model.Token, err error)

	// membership / account & user functions
	CreateUserAccount(dbName, email string) (id string, err error)
	CreateUserToken(dbName string, tok model.Token) (id string, err error)
	SetPasswordResetCode(dbName, tokenID, code string) error
	ResetPassword(dbName, email, code, password string) error
	SetUserRole(dbName, email string, role int) error
	UserSetPassword(dbName, tokenID, password string) error

	// base CRUD
	CreateDocument(auth model.Auth, dbName, col string, doc map[string]interface{}) (map[string]interface{}, error)
	BulkCreateDocument(auth model.Auth, dbName, col string, docs []interface{}) error
	ListDocuments(auth model.Auth, dbName, col string, params model.ListParams) (model.PagedResult, error)
	QueryDocuments(auth model.Auth, dbName, col string, filter map[string]interface{}, params model.ListParams) (model.PagedResult, error)
	GetDocumentByID(auth model.Auth, dbName, col, id string) (map[string]interface{}, error)
	UpdateDocument(auth model.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error)
	UpdateDocuments(auth model.Auth, dbName, col string, filters map[string]interface{}, updateFields map[string]interface{}) (int64, error)
	IncrementValue(auth model.Auth, dbName, col, id, field string, n int) error
	DeleteDocument(auth model.Auth, dbName, col, id string) (int64, error)
	ListCollections(dbName string) ([]string, error)
	ParseQuery(clauses [][]interface{}) (map[string]interface{}, error)

	// form functions
	AddFormSubmission(dbName, form string, doc map[string]interface{}) error
	ListFormSubmissions(dbName, name string) ([]map[string]interface{}, error)
	GetForms(dbName string) ([]string, error)

	// Function functions
	AddFunction(dbName string, data model.ExecData) (string, error)
	UpdateFunction(dbName, id, code, trigger string) error
	GetFunctionForExecution(dbName, name string) (model.ExecData, error)
	GetFunctionByID(dbName, id string) (model.ExecData, error)
	GetFunctionByName(dbName, name string) (model.ExecData, error)
	ListFunctions(dbName string) ([]model.ExecData, error)
	ListFunctionsByTrigger(dbName, trigger string) ([]model.ExecData, error)
	DeleteFunction(dbName, name string) error
	RanFunction(dbName, id string, rh model.ExecHistory) error

	// schedule tasks
	ListTasks() ([]model.Task, error)

	// Files / storage
	AddFile(dbName string, f model.File) (id string, err error)
	GetFileByID(dbName, fileID string) (f model.File, err error)
	DeleteFile(dbName, fileID string) error
	ListAllFiles(dbName, accountID string) ([]model.File, error)
}
