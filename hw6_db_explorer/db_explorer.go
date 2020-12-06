package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	limitParam  = "limit"
	offsetParam = "offset"
)

var defaultParamValues = map[string]string{
	limitParam:  "5",
	offsetParam: "0",
}

type DbApi struct {
	DB         *sql.DB
	TablesMeta map[string]TableData
	TablesList []string
}

type TableData struct {
	Table  string
	Fields map[string]FieldMetaData
	PK     string
}

type FieldMetaData struct {
	Field      string
	Type       string
	Collation  sql.NullString
	Null       string
	Key        string
	Default    sql.NullString
	Extra      string
	Privileges string
	Comment    string
}

type Record map[string]interface{}

type pkHandlingStrategy int

const (
	// IgnorePrimaryKey pk autoincrement игнорируется при вставке
	IgnorePrimaryKey pkHandlingStrategy = iota
	// ThrowErrorOnPrimaryKey primary key нельзя обновлять у существующей записи
	ThrowErrorOnPrimaryKey pkHandlingStrategy = iota
)

type ResponseError struct {
	message    string
	statusCode int
}

func (e ResponseError) Error() string { return e.message }

func NewValidationError(field string) *ResponseError {
	return &ResponseError{
		message:    fmt.Sprintf("field %s have invalid type", field),
		statusCode: http.StatusBadRequest,
	}
}

func NewNotFoundError(message string) *ResponseError {
	return &ResponseError{
		message:    fmt.Sprintf(message),
		statusCode: http.StatusNotFound,
	}
}

func NewDbExplorer(db *sql.DB) (*DbApi, error) {
	return initDbApi(db), nil
}

func initDbApi(db *sql.DB) *DbApi {
	db, err := sql.Open("mysql", DSN)
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	tables, err := db.Query("SHOW TABLES;")
	if err != nil {
		panic(err)
	}

	result := make(map[string]TableData)
	tablesList := make([]string, 0)
	for tables.Next() {
		var tableName string
		tables.Scan(&tableName)
		query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s;", tableName)
		fields, err := db.Query(query)
		if err != nil {
			panic(err)
		}

		fieldDataMap := make(map[string]FieldMetaData)
		var PK string
		for fields.Next() {
			c := FieldMetaData{}
			err = fields.Scan(&c.Field, &c.Type, &c.Collation, &c.Null, &c.Key, &c.Default, &c.Extra, &c.Privileges, &c.Comment)
			if c.Key == "PRI" {
				PK = c.Field
			}
			if err != nil {
				fmt.Print(err)
			}
			fieldDataMap[c.Field] = c
		}
		result[tableName] = TableData{Table: tableName, Fields: fieldDataMap, PK: PK}
		tablesList = append(tablesList, tableName)
	}

	sort.Strings(tablesList)
	return &DbApi{
		DB:         db,
		TablesMeta: result,
		TablesList: tablesList,
	}
}

type Result struct {
	Response interface{} `json:"response,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type Request struct {
	Table string
	ID    string
}

func (h *DbApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res, err := h.handleRequest(r)
	if err != nil {
		if e, ok := err.(*ResponseError); ok {
			data, err := json.Marshal(Result{Error: e.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(e.statusCode)
			w.Write(data)
			return
		}
	}

	data, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (h *DbApi) isTableExist(table string) bool {
	if _, ok := h.TablesMeta[table]; !ok {
		return false
	}

	return true
}

func (h *DbApi) handleRequest(r *http.Request) (*Result, error) {
	req := parseUrlPath(r.URL.Path)
	if req.Table != "" && !h.isTableExist(req.Table) {
		return nil, NewNotFoundError("unknown table")
	}

	pk := h.TablesMeta[req.Table].PK
	switch r.Method {
	case http.MethodGet:
		if req.Table == "" { // route is "/"
			return h.handleGetTables(), nil
		}

		if req.ID == "" { // route is "/$table"
			limit, offset := parseQueryParams(r)
			result, err := h.handleGetRecords(req.Table, limit, offset)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		return h.handleGetById(req.Table, req.ID, pk) // route is "/$table/$id"
	case http.MethodPost:
		body, err := parseRequestBody(r)
		if err != nil {
			return nil, err
		}

		fields, values, err := parseFieldValues(h.TablesMeta[req.Table].Fields, body, ThrowErrorOnPrimaryKey, false)
		if err != nil {
			return nil, err
		}

		return h.handlePost(req.Table, req.ID, pk, fields, values)
	case http.MethodPut:
		body, err := parseRequestBody(r)
		if err != nil {
			return nil, err
		}

		fields, values, err := parseFieldValues(h.TablesMeta[req.Table].Fields, body, IgnorePrimaryKey, true)
		if err != nil {
			return nil, err
		}

		return h.handlePut(req.Table, pk, fields, values)
	case http.MethodDelete:
		return h.handleDelete(req.Table, req.ID, pk)
	}

	//todo error type
	return nil, fmt.Errorf("Bad request")
}

func parseUrlPath(URL string) *Request {
	urlParts := strings.Split(URL, "/")
	var table, id string
	if len(urlParts) > 1 {
		table = urlParts[1]
	}
	if len(urlParts) > 2 {
		id = urlParts[2]
	}
	return &Request{Table: table, ID: id}
}

func parseQueryParams(r *http.Request) (limit string, offset string) {
	limit = r.FormValue(limitParam)
	offset = r.FormValue(offsetParam)
	_, err := strconv.Atoi(limit)
	if err != nil {
		limit = defaultParamValues[limitParam]
	}
	_, err = strconv.Atoi(offset)
	if err != nil {
		offset = defaultParamValues[offsetParam]
	}
	return
}

func parseRequestBody(r *http.Request) (map[string]interface{}, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)
	return data, nil
}

func isInt(f float64) bool {
	return f == float64(int64(f))
}

func (h *DbApi) handleGetTables() *Result {
	return &Result{Response: map[string][]string{"tables": h.TablesList}}
}

func (h *DbApi) handleGetRecords(table, limit, offset string) (*Result, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %s, %s", table, offset, limit)
	records, err := executeSelectQuery(h.DB, h.TablesMeta[table].Fields, query)
	if err != nil {
		return nil, err
	}
	return &Result{Response: map[string][]Record{"records": records}}, nil // todo create type for this
}

func (h *DbApi) handleGetById(table, id, pk string) (*Result, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?;", table, pk)
	params := []interface{}{id}
	records, err := executeSelectQuery(h.DB, h.TablesMeta[table].Fields, query, params...)
	if err != nil {
		return nil, err
	}
	return &Result{Response: Record{"record": records[0]}}, nil
}

func (h *DbApi) handlePut(table, pk string, fields []string, values []interface{}) (*Result, error) {
	placeholders := make([]string, len(fields))
	for i := 0; i < len(fields); i++ {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(fields, ","), strings.Join(placeholders, ","))
	rows, err := h.DB.Exec(query, values...)
	if err != nil {
		fmt.Println(err.Error())
	}

	lastInsertId, err := rows.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Result{Response: map[string]int64{pk: lastInsertId}}, nil
}

func (h *DbApi) handlePost(table, id, pk string, fields []string, values []interface{}) (*Result, error) {
	sets := ""
	for i := 0; i < len(fields); i++ {
		comma := ","
		if i == len(fields)-1 {
			comma = ""
		}
		// todo something
		if values[i] == nil {
			sets += fmt.Sprintf("%s = %s%s", fields[i], "NULL", comma)
		} else {
			sets += fmt.Sprintf("%s = \"%s\"%s", fields[i], values[i], comma) //todo use stringbuilder
		}
	}

	queryParams := []interface{}{id}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, sets, pk)
	rows, err := h.DB.Exec(query, queryParams...)
	if err != nil {
		fmt.Println(err.Error())
	}

	updated, err := rows.RowsAffected()
	if err != nil {
		return nil, err
	}

	return &Result{Response: map[string]int64{"updated": updated}}, nil
}

func (h *DbApi) handleDelete(table, id, pk string) (*Result, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s", table, pk, id)
	rows, err := h.DB.Exec(query)
	if err != nil {
		return nil, err
	}
	deleted, err := rows.RowsAffected()
	if err != nil {
		return nil, err
	}

	return &Result{Response: map[string]int64{"deleted": deleted}}, nil
}

func executeSelectQuery(DB *sql.DB, columns map[string]FieldMetaData, query string, params ...interface{}) ([]Record, error) {
	rows, err := DB.Query(query, params...)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0)
	colTypes, err := rows.ColumnTypes()

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		cols := make([]interface{}, len(columns))
		for i := range cols {
			if strings.Contains(columns[colTypes[i].Name()].Type, "int") {
				cols[i] = new(*int64)
			} else {
				cols[i] = new(*string)
			}
		}

		err = rows.Scan(cols...)
		if err != nil {
			return nil, err
		}

		colByName := make(map[string]interface{}, len(columns))
		for i, col := range cols {
			colByName[colTypes[i].Name()] = col
		}

		records = append(records, colByName)
	}

	if len(records) == 0 {
		return nil, NewNotFoundError("record not found")
	}

	return records, nil
}

func parseFieldValues(meta map[string]FieldMetaData, fieldValues map[string]interface{}, pkStrategy pkHandlingStrategy, setDefaultValueIfEmpty bool) (fields []string, values []interface{}, err error) {
	fields = make([]string, 0)
	values = make([]interface{}, 0)

	for field, fmeta := range meta {
		value, inRequest := fieldValues[field]

		if inRequest && fmeta.Key == "PRI" {
			if fmeta.Extra == "auto_increment" && pkStrategy == IgnorePrimaryKey {
				continue
			}
			if pkStrategy == ThrowErrorOnPrimaryKey {
				return nil, nil, NewValidationError(field)
			}
		}

		if !inRequest && fmeta.Null == "NO" && setDefaultValueIfEmpty {
			fields = append(fields, field)
			if strings.Contains(fmeta.Type, "varchar") || strings.Contains(fmeta.Type, "text") {
				values = append(values, "")
			} else if strings.Contains(fmeta.Type, "int") || strings.Contains(fmeta.Type, "float") {
				values = append(values, 0)
			}
			continue
		} else if inRequest {
			if !validateField(value, fmeta) {
				return nil, nil, NewValidationError(fmeta.Field)
			}

			fields = append(fields, field)
			values = append(values, value)
		}
	}

	return
}

func validateField(value interface{}, meta FieldMetaData) bool {
	switch v := value.(type) {
	case string:
		if !strings.Contains(meta.Type, "text") && !strings.Contains(meta.Type, "varchar") {
			return false
		}
	case float64: // int is unpacked as float also
		if strings.Contains(meta.Type, "int") && !isInt(v) {
			return false
		} else if !strings.Contains(meta.Type, "float") && !strings.Contains(meta.Type, "double") {
			return false
		}
	case nil:
		if strings.Contains(meta.Null, "YES") {
			return true
		}
		return false
	default:
		// todo
	}

	return true
}
