package country

import (
	"database/sql"
	"fmt"
)

var countryColumns = []string{
	"countrycode",
	"countryname",
	"code",
}

func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS country (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			` + createColumnDDL() + `
		)
	`)
	if err != nil {
		return err
	}

	for _, column := range countryColumns {
		stmt := fmt.Sprintf("ALTER TABLE country ADD COLUMN IF NOT EXISTS %s TEXT NULL", quoteColumn(column))
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func createColumnDDL() string {
	return "`countrycode` VARCHAR(3) NOT NULL,\n\t\t\t`countryname` VARCHAR(200) NOT NULL,\n\t\t\t`code` CHAR(2) NULL"
}

func quoteColumn(name string) string {
	return "`" + name + "`"
}
