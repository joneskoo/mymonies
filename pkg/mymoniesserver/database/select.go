package database

import "strings"

type SelectQuery struct {
	Columns []string
	From    string
	Where   []string
	GroupBy string
	OrderBy string
}

func (q SelectQuery) SQL() string {
	columns := "SELECT " + strings.Join(q.Columns, ", ")
	var from, where, groupBy, orderBy string
	if len(q.From) > 0 {
		from = " FROM " + q.From
	}
	if len(q.Where) > 0 {
		where = " WHERE " + strings.Join(q.Where, " AND ")
	}
	if len(q.GroupBy) > 0 {
		groupBy = " GROUP BY " + q.GroupBy
	}
	if len(q.OrderBy) > 0 {
		orderBy = " ORDER BY " + q.OrderBy
	}
	return strings.Join([]string{columns, from, where, groupBy, orderBy}, "")
}

func (q *SelectQuery) AndWhere(cond string) {
	q.Where = append(q.Where, cond)
}
