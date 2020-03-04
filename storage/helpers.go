package storage

import "strings"

// buildInsertQuery creates an INSERT query string suitable for use as a prepared statement.
func buildInsertQuery(table string, columns []string, ignore bool) string {
	b := strings.Builder{}
	b.WriteString("INSERT ")
	if ignore {
		b.WriteString("OR IGNORE ")
	}
	b.WriteString("INTO ")
	b.WriteString(table)
	b.WriteString(" (")

	insertColumns := strings.Join(columns[1:],",")
	b.WriteString(insertColumns)
	b.WriteString(" ) VALUES (")

	insertPlaceholders := strings.Repeat("?,",len(columns)-2)
	b.WriteString(insertPlaceholders)
	b.WriteString("?)")
	return b.String()
}

// buildUpdateQuery creates an UPDATE query string suitable for use as a prepared statement.
func buildUpdateQuery(table string, columns []string) string {
	b := strings.Builder{}
	b.WriteString("UPDATE ")
	b.WriteString(table)
	b.WriteString(" SET ")

	first := true
	for _, column := range columns[1:] {
		if first {
			first = false
		} else {
			b.WriteByte(',')
		}
		b.WriteString(column)
		b.WriteString("=?")
	}
	b.WriteString(" WHERE id=?")
	return b.String()
}

func buildSelectQuery(table string, columns []string) (query, count string) {
	b := strings.Builder{}
	b.WriteString("SELECT ")
	first := true
	for _, column := range columns {
		if first {
			first = false
		} else {
			b.WriteByte(',')
		}
		b.WriteString(column)
	}
	b.WriteString(" FROM ")
	b.WriteString(table)
	query = b.String()

	b.Reset()
	b.WriteString("SELECT COUNT(*) FROM ")
	b.WriteString(table)
	count = b.String()

	return
}

// Takes a table name and a list of column names, and prepends table+"." to each column name.
func getColumnsWithTable(table string, columns []string) []string {
	r := make([]string,len(columns))

	t := table + "."
	for k, column := range columns {
		r[k] = t + column
	}
	return r
}