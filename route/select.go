package route

import (
	"strings"

	"github.com/berkaroad/saashard/net/mysql"
	"github.com/berkaroad/saashard/sqlparser"
)

var currentUserField = &mysql.Field{Schema: []byte(""),
	Table:        []byte(""),
	OrgTable:     []byte(""),
	Name:         []byte("current_user()"),
	OrgName:      []byte(""),
	Charset:      uint16(mysql.DEFAULT_COLLATION_ID),
	ColumnLength: 423,
	ColumnType:   mysql.MYSQL_TYPE_VAR_STRING,
	Flags:        mysql.NOT_NULL_FLAG,
	Decimals:     31}

var versionField = &mysql.Field{Schema: []byte(""),
	Table:        []byte(""),
	OrgTable:     []byte(""),
	Name:         []byte("version()"),
	OrgName:      []byte(""),
	Charset:      uint16(mysql.DEFAULT_COLLATION_ID),
	ColumnLength: 72,
	ColumnType:   mysql.MYSQL_TYPE_VAR_STRING,
	Flags:        mysql.NOT_NULL_FLAG,
	Decimals:     31}

var connectionIDField = &mysql.Field{Schema: []byte(""),
	Table:        []byte(""),
	OrgTable:     []byte(""),
	Name:         []byte("CONNECTION_ID()"),
	OrgName:      []byte(""),
	Charset:      uint16(mysql.DEFAULT_COLLATION_ID),
	ColumnLength: 10,
	ColumnType:   mysql.MYSQL_TYPE_LONGLONG,
	Flags:        mysql.NOT_NULL_FLAG | mysql.BINARY_FLAG,
	Decimals:     0}

var databaseField = &mysql.Field{Schema: []byte(""),
	Table:        []byte(""),
	OrgTable:     []byte(""),
	Name:         []byte("DATABASE()"),
	OrgName:      []byte(""),
	Charset:      uint16(mysql.DEFAULT_COLLATION_ID),
	ColumnLength: 102,
	ColumnType:   mysql.MYSQL_TYPE_VAR_STRING,
	Flags:        mysql.NOT_NULL_FLAG,
	Decimals:     31}

func (r *Router) buildSimpleSelectPlan(statement *sqlparser.SimpleSelect) (*Plan, error) {
	schemaConfig := r.Schemas[r.SchemaName]
	supportedFieldNames := map[string]*mysql.Field{
		"current_user()":  currentUserField,
		"version()":       versionField,
		"connection_id()": connectionIDField,
		"database()":      databaseField}

	supportedFieldValues := map[string]func(*mysql.Row){
		"current_user()":  func(row *mysql.Row) { row.AppendStringValue(r.User) },
		"version()":       func(row *mysql.Row) { row.AppendStringValue(mysql.ServerVersion) },
		"connection_id()": func(row *mysql.Row) { row.AppendUIntValue(uint64(r.ConnectionID)) },
		"database()":      func(row *mysql.Row) { row.AppendStringValue(r.SchemaName) }}

	allFieldsSupported := true
	for _, fieldExpr := range statement.SelectExprs {
		fieldName := strings.ToLower(sqlparser.String(fieldExpr))
		if _, ok := supportedFieldNames[fieldName]; !ok {
			allFieldsSupported = false
			break
		}
	}

	plan := new(Plan)

	plan.DataNode = schemaConfig.Nodes[0]
	plan.IsSlave = true
	plan.Statement = statement

	if allFieldsSupported {
		result := new(mysql.Result)
		result.Status = mysql.SERVER_STATUS_AUTOCOMMIT
		result.Resultset = new(mysql.Resultset)
		result.Resultset.Fields = make([]*mysql.Field, len(statement.SelectExprs))
		for i, fieldExpr := range statement.SelectExprs {
			fieldName := strings.ToLower(sqlparser.String(fieldExpr))
			result.Resultset.Fields[i] = supportedFieldNames[fieldName]
		}
		result.Rows = make([]*mysql.Row, 1)
		row := mysql.NewTextRow(result.Resultset.Fields)
		for _, fieldExpr := range statement.SelectExprs {
			fieldName := strings.ToLower(sqlparser.String(fieldExpr))
			supportedFieldValues[fieldName](row)
		}
		result.Rows[0] = row
		plan.Result = result
	}
	return plan, nil
}

func (r *Router) buildSelectPlan(statement *sqlparser.Select) (*Plan, error) {
	schemaConfig := r.Schemas[r.SchemaName]
	plan := new(Plan)

	plan.DataNode = schemaConfig.Nodes[0]
	plan.IsSlave = true
	plan.Statement = statement

	return plan, nil
}
