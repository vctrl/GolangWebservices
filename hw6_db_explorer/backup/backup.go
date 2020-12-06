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

func NewDbExplorer(db *sql.DB) (*DbApi, error) {
	return initDbApi(db), nil
}

type DbApi struct {
	DB     *sql.DB
	Tables map[string]map[string]ColumnData
}

type ColumnData struct {
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

type Table struct {
	Name string
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

	result := make(map[string]map[string]ColumnData)
	for tables.Next() {
		t := Table{}
		tables.Scan(&t.Name)
		query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s;", t.Name)
		cols, err := db.Query(query)
		if err != nil {

		}

		colDataMap := make(map[string]ColumnData)
		for cols.Next() {
			c := ColumnData{}
			err = cols.Scan(&c.Field, &c.Type, &c.Collation, &c.Null, &c.Key, &c.Default, &c.Extra, &c.Privileges, &c.Comment)
			if err != nil {
				fmt.Print(err)
			}
			colDataMap[c.Field] = c
		}
		result[t.Name] = colDataMap
	}

	return &DbApi{
		DB:     db,
		Tables: result,
	}
}

type Result struct {
	Response interface{} `json:"response,omitempty"`
	Error    string      `json:"error,omitempty"`
}

func (h *DbApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		tables := FormatTables(h.Tables)
		data, err := json.Marshal(tables)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(data)
	default:
		urlParts := strings.Split(r.URL.Path, "/")
		table := urlParts[1]
		tableMetaData, ok := h.Tables[table]
		if !ok {
			result := Result{Error: "unknown table"}
			data, err := json.Marshal(result)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write(data)
			return
		}
		switch r.Method {
		case "GET":
			if len(urlParts) == 3 {
				id := urlParts[2]
				pk := getPrimaryKeyField(h.Tables, table)
				rows, err := h.HandleTableIdGet(table, pk, id)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				var result Result
				if len(rows) == 0 {
					w.WriteHeader(http.StatusNotFound)
					result = Result{Error: "record not found"}
				} else {
					item := make(map[string]map[string]interface{})
					item["record"] = rows[0]
					result = Result{Response: item}
				}

				data, err := json.Marshal(result)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Write(data)
				return
			}

			limit := r.FormValue("limit")
			offset := r.FormValue("offset")
			_, err := strconv.Atoi(limit)
			if err != nil {
				limit = "5"
			}
			_, err = strconv.Atoi(offset)
			if err != nil {
				offset = "0"
			}
			// todo refactor repeating code
			rows, err := h.HandleTableGet(table, limit, offset)
			items := make(map[string][]map[string]interface{})
			items["records"] = rows
			result := Result{Response: items}
			data, err := json.Marshal(result)
			if err != nil {

			}
			w.Write(data)

		case "PUT":
			// todo refactor repeating code
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			var data map[string]interface{}
			json.Unmarshal(body, &data)
			fields, values, err := Validate(tableMetaData, data, r.Method)
			// todo refactoring
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				result := Result{Error: err.Error()}
				data, err := json.Marshal(result)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Write(data)
				return
			}
			pk := getPrimaryKeyField(h.Tables, table)
			result, err := h.HandleTablePut(w, r, table, pk, fields, values)
			if err != nil {

			}
			w.Write(result)
		case "POST":
			if len(urlParts) == 3 {
				id := urlParts[2]
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				var data map[string]interface{}
				json.Unmarshal(body, &data)
				fields, values, err := Validate(tableMetaData, data, r.Method)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					result := Result{Error: err.Error()}
					data, err := json.Marshal(result)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.Write(data)
					return
				}
				pk := getPrimaryKeyField(h.Tables, table)
				result, err := h.HandleTableIdPost(w, r, table, fields, values, pk, id)
				if err != nil {

				}
				w.Write(result)
			} else {
				//todo badrequest
			}
		case "DELETE":
			if len(urlParts) == 3 {
				id := urlParts[2]
				pk := getPrimaryKeyField(h.Tables, table)
				data, err := h.HandleTableIdDelete(w, r, table, pk, id)
				if err != nil {

				}
				w.Write(data)
			} else {
				//todo badrequest
			}
		}
	}
}

func FormatTables(input map[string]map[string]ColumnData) Result {
	tables := make([]string, len(input))
	i := 0
	for t := range input {
		tables[i] = t
		i++
	}

	sort.Strings(tables)
	result := Result{}
	response := make(map[string][]string)
	response["tables"] = tables
	result.Response = response
	return result
}

func (h *DbApi) HandleTableGet(table, limit, offset string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %s, %s", table, offset, limit)
	rows, err := executeSelectQuery(h.DB, query, h.Tables[table])
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// todo rename
func (h *DbApi) HandleTableIdGet(table string, pk string, id string) ([]map[string]interface{}, error) {
	queryParams := []interface{}{id}
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?;", table, pk)
	rows, err := executeSelectQuery(h.DB, query, h.Tables[table], queryParams...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func executeSelectQuery(db *sql.DB, query string, meta map[string]ColumnData, params ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, err
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	records := make([]map[string]interface{}, 0)
	for rows.Next() {
		cols := make([]interface{}, len(colTypes))    // array of column values
		colPtrs := make([]interface{}, len(colTypes)) // array of pointers for rows.Scan() function
		for i := 0; i < len(colPtrs); i++ {
			colPtrs[i] = &cols[i]
		}

		err = rows.Scan(colPtrs...)
		if err != nil {

		}
		colByName := make(map[string]interface{}, len(colTypes))
		for i, colType := range colTypes {
			var value interface{}

			switch cols[i].(type) {
			case int:
				value, err = strconv.Atoi(string(cols[i].([]byte)))
				if err != nil {
					return nil, err
				}
			case []byte:
				valueStr := string(cols[i].([]byte))
				// cons:
				// 1. excessive casts
				// 2. using colType.Name()
				if strings.Contains(meta[colType.Name()].Type, "int") {
					value, err = strconv.Atoi(valueStr)
				} else {
					value = valueStr
				}
				if err != nil {
					return nil, err
				}
			default:
				value = cols[i]
			}

			colByName[colType.Name()] = value
		}

		records = append(records, colByName)
	}

	return records, nil
}

// todo // зависит от типа запроса: если post и нет значения,

//если put то проставлять дефолтные значения
// если post то просто обновлять

func Validate(meta map[string]ColumnData, inputData map[string]interface{}, rMethod string) ([]string, []interface{}, error) {
	fields := make([]string, 0)
	values := make([]interface{}, 0)

	// если put то надо все поля проходить и дефолтные значения проставлять
	// если post то просто обновляем поля которые нужно обновить и всё

	for m, meta := range meta {
		pk := meta.Key == "PRI"
		_, inRequest := inputData[m]
		if inRequest && pk && rMethod == "POST" {
			// primary key нельзя обновлять у существующей записи
			return nil, nil, fmt.Errorf("field %s have invalid type", m)
		} else if pk && rMethod == "POST" {
			continue
		} else if rMethod == "PUT" && pk && strings.Contains(meta.Extra, "auto_increment") {
			// auto increment primary key игнорируется при вставке
			continue
		} else if rMethod == "POST" && !inRequest {
			continue
		}

		fields = append(fields, m)

		// логика только для put
		if !inRequest && meta.Null == "NO" {
			// depending on type set default value!
			if strings.Contains(meta.Type, "varchar") || strings.Contains(meta.Type, "text") {
				values = append(values, "")
			} else if strings.Contains(meta.Type, "int") || strings.Contains(meta.Type, "float") {
				values = append(values, 0)
			}

			continue
		}

		switch v := inputData[m].(type) {
		case string:
			if !strings.Contains(meta.Type, "text") && !strings.Contains(meta.Type, "varchar") {
				return nil, nil, fmt.Errorf("field %s have invalid type", m)
			}
			values = append(values, v)
		case float64: // int is unpacked as float also
			if strings.Contains(meta.Type, "int") {
				if v == float64(int64(v)) {
					values = append(values, strconv.Itoa(int(v)))
				} else {
					return nil, nil, fmt.Errorf("field %s have invalid type", m)
				}
			} else if strings.Contains(meta.Type, "float") || strings.Contains(meta.Type, "double") {
				values = append(values, strconv.FormatFloat(v, 'f', 6, 64))
			} else {
				return nil, nil, fmt.Errorf("field %s have invalid type", m)
			}
		case nil:
			// зависит есть в запросе или нет
			if inRequest && meta.Null == "NO" {
				return nil, nil, fmt.Errorf("field %s have invalid type", m)
			} else {
				values = append(values, nil)
			}
		default:
			// todo
		}
	}

	return fields, values, nil
}
func (h *DbApi) HandleTablePut(w http.ResponseWriter, r *http.Request, table string, pk string, fields []string, values []interface{}) ([]byte, error) {
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

	}

	content := make(map[string]int64)
	content[pk] = lastInsertId
	result := Result{Response: content}
	data, err := json.Marshal(result)
	if err != nil {

	}

	return data, nil
}

// HandlePostData
func (h *DbApi) HandleTableIdPost(w http.ResponseWriter, r *http.Request, table string, fields []string, values []interface{}, pk string, id string) ([]byte, error) {
	sets := ""
	for i := 0; i < len(fields); i++ {
		comma := ","
		if i == len(fields)-1 {
			comma = ""
		}
		// todo something )))))))))0
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

	//todo refactor
	updated, err := rows.RowsAffected()
	if err != nil {

	}
	content := make(map[string]int64)
	content["updated"] = updated
	result := Result{Response: content}
	data, err := json.Marshal(result)
	if err != nil {

	}
	return data, nil
}

// todo HandleDeleteDataById
func (h *DbApi) HandleTableIdDelete(w http.ResponseWriter, r *http.Request, table string, pk string, id string) ([]byte, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s", table, pk, id)
	rows, err := h.DB.Exec(query)
	if err != nil {

	}
	deleted, err := rows.RowsAffected()
	if err != nil {

	}
	content := make(map[string]int64)
	content["deleted"] = deleted
	result := Result{Response: content}
	data, err := json.Marshal(result)
	if err != nil {

	}
	return data, nil
}

func getPrimaryKeyField(tables map[string]map[string]ColumnData, table string) string {
	for _, meta := range tables[table] {
		if meta.Key == "PRI" {
			return meta.Field
		}
	}
	return ""
}
