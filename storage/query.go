package storage

import (
	"strings"
	"fmt"
	"strconv"
	)

type queryFilter struct {
	value string
	exact bool
	set   bool
}
type querySort struct {
	value     string
	ascending bool
}

type Query struct {
	sortValues []querySort
	filters    map[string]queryFilter
	multi      []map[string]queryFilter
	offset     int
	limit      int

	random        bool
	modified      bool
	cachedClauses string
	cachedValues []interface{}
}

func NewQuery() *Query {
	return &Query{
		sortValues: make([]querySort, 0, 8),
		filters:    make(map[string]queryFilter),
		multi:      make([]map[string]queryFilter, 0, 8),
		modified:   true,
	}
}

func (q *Query) Random() *Query {
	q.random = true
	return q
}

func (q *Query) SortedBy(value string, ascending bool) *Query {
	if value == "" {
		return q
	}
	q.sortValues = append(q.sortValues, querySort{
		value:     value,
		ascending: ascending,
	})
	q.modified = true
	return q
}

func (q *Query) Filtered(key string, value string, exact bool) *Query {
	if key == "" {
		return q
	}
	q.filters[key] = queryFilter{
		value: value,
		exact: exact,
	}
	q.modified = true
	return q
}

func (q *Query) OrFiltered(value string, exact bool, keys ...string) *Query {
	if len(keys) == 0 {
		return q
	}

	multi := make(map[string]queryFilter)
	for _, key := range keys {
		multi[key] = queryFilter{
			value: value,
			exact: exact,
		}
	}

	q.multi = append(q.multi, multi)
	q.modified = true
	return q
}

func (q *Query) In(key string, values []int) *Query {
	strVals := make([]string, len(values))
	for k, v := range values {
		strVals[k] = strconv.Itoa(v)
	}
	q.filters[key] = queryFilter{
		value: strings.Join(strVals, ","),
		set:   true,
	}
	q.modified = true
	return q
}

func (q *Query) Skip(n int) *Query {
	q.offset = n
	return q
}

func (q *Query) Take(n int) *Query {
	q.limit = n
	return q
}

var queryExactStrings = map[bool]string{
	false: "(%s LIKE ?)",
	true:  "(%s = ?)",
}
var querySortAscending = map[bool]string{
	false: "DESC",
	true:  "ASC",
}

func (q *Query) buildCountSelect(baseQuery string, validColumns []string) (queryString string, bindValues []string, err error) {
	return
}

func (q *Query) buildSelect(baseQuery string, validColumns []string) (queryString string, bindValues []interface{}, err error) {

	b := strings.Builder{}
	b.WriteString(baseQuery)

	if q.modified {
		cachedValues := make([]interface{},0,len(q.filters))
		// validate filter/sort column names in case they're from untrusted input
		isValidColumn := func(col string) bool {
			for _, column := range validColumns {
				if col == column {
					return true
				}
			}
			return false
		}
		writeFilter := func(w *strings.Builder, column string, f queryFilter) {
			if f.set {
				// value.set can only be assigned from In() which only accepts an integer slice, so the next line
				// should be safe from SQL injection
				w.WriteString(fmt.Sprintf("%s IN (%s)", column, f.value))
			} else {
				w.WriteString(fmt.Sprintf(queryExactStrings[f.exact], column))
				keyword := f.value
				if !f.exact {
					keyword = "%" + keyword + "%"
				}
				cachedValues = append(cachedValues, keyword)
			}
		}

		wroteWhere := strings.Contains(baseQuery, " WHERE ")

		c := strings.Builder{}
		if len(q.filters) > 0 {
			if wroteWhere {
				c.WriteString(" AND ")
			} else {
				c.WriteString(" WHERE ")
				wroteWhere = true
			}

			first := true
			for key, value := range q.filters {
				if !isValidColumn(key) {
					err = fmt.Errorf("invalid column name \"%s\"", key)
					return
				}

				if first {
					first = false
				} else {
					c.WriteString(" AND ")
				}

				writeFilter(&c, key, value)
			}
		}

		if len(q.multi) > 0 {
			if wroteWhere {
				c.WriteString(" AND ")
			} else {
				c.WriteString(" WHERE ")
			}
			c.WriteByte('(')

			first := true
			for _, multi := range q.multi {
				if first {
					first = false
				} else {
					c.WriteString(" AND ")
				}

				mfirst := true
				for key, value := range multi {
					if mfirst {
						mfirst = false
					} else {
						c.WriteString(" OR ")
					}
					writeFilter(&c, key, value)
				}

			}

		}

		if len(q.sortValues) > 0 || q.random {
			c.WriteString(" ORDER BY ")

			if q.random {
				c.WriteString("RANDOM() ")
			} else {
				first := true
				for _, sortV := range q.sortValues {
					if !isValidColumn(sortV.value) {
						err = fmt.Errorf("invalid column name \"%s\"", sortV.value)
						return
					}

					if first {
						first = false
					} else {
						c.WriteByte(',')
					}

					c.WriteString(sortV.value + " " + querySortAscending[sortV.ascending])
				}
			}
		}

		q.modified = false
		q.cachedClauses = c.String()
		q.cachedValues = cachedValues
	}
	b.WriteString(q.cachedClauses)

	bindValues = make([]interface{},0,len(q.cachedValues))
	bindValues = append(bindValues,q.cachedValues...)

	if q.offset > 0 || q.limit > 0 {
		b.WriteString(" LIMIT ")
		if q.offset > 0 {
			b.WriteString(strconv.Itoa(q.offset) + ",")
		}
		if q.limit > 0 || q.offset > 0 {
			b.WriteString(strconv.Itoa(q.limit))
		}
	}

	queryString = b.String()
	return
}
