package gpa

// Dialect constants
const (
	DialectSQLite = "sqlite"
	DialectMySQL  = "mysql"
	DialectPgSQL  = "pgsql"
	DialectMsSQL  = "mssql"
)

// SupportedDialects is a list of all supported database dialects
var SupportedDialects = []string{
	DialectSQLite,
	DialectMySQL,
	DialectPgSQL,
	DialectMsSQL,
}

// IsDialectSupported checks if the given dialect is supported
func IsDialectSupported(dialect string) bool {
	for _, d := range SupportedDialects {
		if d == dialect {
			return true
		}
	}
	return false
}
