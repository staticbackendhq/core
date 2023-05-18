package sqlite

type Scanner interface {
	Scan(dest ...interface{}) error
}
