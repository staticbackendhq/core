package internal

const (
	DataStorePostgreSQL = "postgresql"
	DataStoreMongoDB    = "mongo"
)

type Persister interface {
	Ping() error

	// customer / app related
	CreateCustomer(Customer) (Customer, error)
	CreateBase(BaseConfig) (BaseConfig, error)
	EmailExists(email string) (bool, error)
	FindAccount(customerID string) (Customer, error)
	FindDatabase(baseID string) (BaseConfig, error)
	DatabaseExists(name string) (bool, error)
	ListDatabases() ([]BaseConfig, error)
	IncrementMonthlyEmailSent(baseID string) error
	GetCustomerByStripeID(stripeID string) (cus Customer, err error)
	ActivateCustomer(customerID string) error
	NewID() string

	// system user account function s
	FindToken(dbName, tokenID, token string) (Token, error)
	FindRootToken(dbName, tokenID, accountID, token string) (Token, error)
	GetRootForBase(dbName string) (Token, error)
	FindTokenByEmail(dbName, email string) (Token, error)
	UserEmailExists(dbName, email string) (exists bool, err error)
	GetFirstTokenFromAccountID(dbName, accountID string) (tok Token, err error)

	// membership / account & user functions
	CreateUserAccount(dbName, email string) (id string, err error)
	CreateUserToken(dbName string, tok Token) (id string, err error)
	SetPasswordResetCode(dbName, tokenID, code string) error
	ResetPassword(dbName, email, code, password string) error
	SetUserRole(dbName, email string, role int) error
	UserSetPassword(dbName, tokenID, password string) error

	// base CRUD
	CreateDocument(auth Auth, dbName, col string, doc map[string]interface{}) (map[string]interface{}, error)
	BulkCreateDocument(auth Auth, dbName, col string, docs []interface{}) error
	ListDocuments(auth Auth, dbName, col string, params ListParams) (PagedResult, error)
	QueryDocuments(auth Auth, dbName, col string, filter map[string]interface{}, params ListParams) (PagedResult, error)
	GetDocumentByID(auth Auth, dbName, col, id string) (map[string]interface{}, error)
	UpdateDocument(auth Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error)
	IncrementValue(auth Auth, dbName, col, id, field string, n int) error
	DeleteDocument(auth Auth, dbName, col, id string) (int64, error)
	ListCollections(dbName string) ([]string, error)
	ParseQuery(clauses [][]interface{}) (map[string]interface{}, error)

	// form functions
	AddFormSubmission(dbName, form string, doc map[string]interface{}) error
	ListFormSubmissions(dbName, name string) ([]map[string]interface{}, error)
	GetForms(dbName string) ([]string, error)

	// Function functions
	AddFunction(dbName string, data ExecData) (string, error)
	UpdateFunction(dbName, id, code, trigger string) error
	GetFunctionForExecution(dbName, name string) (ExecData, error)
	GetFunctionByID(dbName, id string) (ExecData, error)
	GetFunctionByName(dbName, name string) (ExecData, error)
	ListFunctions(dbName string) ([]ExecData, error)
	ListFunctionsByTrigger(dbName, trigger string) ([]ExecData, error)
	DeleteFunction(dbName, name string) error
	RanFunction(dbName, id string, rh ExecHistory) error

	// schedule tasks
	ListTasks() ([]Task, error)

	// Files / storage
	AddFile(dbName string, f File) (id string, err error)
	GetFileByID(dbName, fileID string) (f File, err error)
	DeleteFile(dbName, fileID string) error
}
