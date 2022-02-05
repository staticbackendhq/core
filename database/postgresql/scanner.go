package postgresql

type Scanner interface {
	Scan(dest ...interface{}) error
}
