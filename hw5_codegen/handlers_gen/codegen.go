package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"
)

type ApiMetaData struct {
	Auth         bool // todo
	Name         string
	Url          string
	Method       string // todo
	ParamsStruct string
}

type ParamMetaData struct {
	Name        string
	Type        string
	Validations string
}

func main() {
	apis := make(map[string][]ApiMetaData)
	params := make(map[string][]ParamMetaData)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line

	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out) // empty line

	for _, d := range node.Decls {
		f, ok := d.(*ast.FuncDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.FuncDecl\n", d)
			continue
		}
		needCodegen := false
		if f.Doc != nil {
			for _, comment := range f.Doc.List {
				needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
			}
			if !needCodegen {
				fmt.Printf("SKIP func %#v doesnt have apigen mark\n", f.Name.Name)
				continue
			}
			if f.Recv != nil {
				if r, rok := f.Recv.List[0].Type.(*ast.StarExpr); rok {
					_, ex := apis[r.X.(*ast.Ident).Name]
					if !ex {
						apis[r.X.(*ast.Ident).Name] = make([]ApiMetaData, 0)
					}
					apiMetaData := ApiMetaData{}
					// todo hmmmm
					for _, comment := range f.Doc.List {
						err := json.Unmarshal([]byte(after(comment.Text, "// apigen:api")), &apiMetaData)
						if err != nil {
							fmt.Println("Can't parse api info")
							continue
						}

						apiMetaData.Name = f.Name.Name
						paramsStructName := f.Type.Params.List[1].Type.(*ast.Ident).Name
						apiMetaData.ParamsStruct = paramsStructName

						// запоминаем имя структуры, чтобы найти на втором проходе
						var paramMetaData []ParamMetaData
						params[paramsStructName] = paramMetaData

						currAPI := apis[r.X.(*ast.Ident).Name]
						currAPI = append(currAPI, apiMetaData)
						apis[r.X.(*ast.Ident).Name] = currAPI
					}
				}
			}
		}
	}

	// second traversal to get parameters structs and fields
	for _, f := range node.Decls {
		g, ok := f.(*ast.GenDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.GenDeckl\n", g)
			continue
		}
		for _, spec := range g.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				fmt.Printf("SKIP %T is not *ast.TypeSpec\n", currType)
				continue
			}
			// если запомнили, то нужно получить данные по этой структуре для парсинга параметров и валидаций
			if currParams, ex := params[currType.Name.Name]; ex {
				if !ex {
					params[currType.Name.Name] = make([]ParamMetaData, 0)
				}
				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %T is not *ast.StructType\n", currStruct)
					continue
				}

				pmd := ParamMetaData{}
				for _, field := range currStruct.Fields.List {
					if field.Tag != nil {
						if tag, ok := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Lookup("apivalidator"); ok {
							pmd.Validations = string(tag)
						}
					}
					pmd.Type = field.Type.(*ast.Ident).Name
					pmd.Name = field.Names[0].Name
					currParams = append(currParams, pmd)
					params[currType.Name.Name] = currParams
				}
			}
		}
	}

	// response type
	fmt.Fprintln(out, `type Response struct {`)
	fmt.Fprintln(out, `Err  string `+"`json:\"error\"`")
	fmt.Fprintln(out, `User interface{}`+"`json:\"response,omitempty\"`")
	fmt.Fprintln(out, `}`)
	fmt.Fprintln(out) // empty line

	writeErrorFunc := `
	func writeError(w http.ResponseWriter, statusCode int, message string) {
		result, err := json.Marshal(Response{Err: message})
		if err != nil {
	
		}
		w.WriteHeader(statusCode)
		w.Write(result)
	}
	`
	fmt.Fprintln(out, writeErrorFunc)
	// ServeHTTP
	for api := range apis {
		fmt.Fprintln(out, `func (h *`+api+`) ServeHTTP(w http.ResponseWriter, r *http.Request) {`)
		fmt.Fprintln(out, `switch r.URL.Path {`)
		for _, endpoint := range apis[api] {
			fmt.Fprintln(out, `case "`+endpoint.Url+`":`)
			fmt.Fprintln(out, `h.handle`+endpoint.Name+`(w, r)`)

			// fmt.Fprintln(out, `func (h *`+structName+` handle`+endpoint.Name+`(w http.ResponseWriter, r *http.Request) {`)
		}

		fmt.Fprintln(out, `default:`)
		fmt.Fprintln(out, `writeError(w, http.StatusNotFound, "unknown method")`)
		fmt.Fprintln(out, `return`)
		fmt.Fprintln(out, `}`)
		fmt.Fprintln(out, `}`)
		fmt.Fprintln(out) // empty line
	}

	// generate methods for every API
	for api := range apis {
		for _, endpoint := range apis[api] {
			fmt.Fprintln(out, `func (h *`+api+`) handle`+endpoint.Name+`(w http.ResponseWriter, r *http.Request) {`)
			// auth check
			authCheckCode := `	if r.Header.Get("X-Auth") != "100500" {
				writeError(w, http.StatusForbidden, "unauthorized")
				return
			}`
			if endpoint.Auth {
				fmt.Fprintln(out, authCheckCode)
			}

			// http method check
			httpMethodCheckCode := `if r.Method != "` + endpoint.Method + `" {
				writeError(w, http.StatusNotAcceptable, "bad method")
				return
			}
			`
			if endpoint.Method != "" {
				fmt.Fprintln(out, httpMethodCheckCode)
			}

			// parse query params
			fmt.Fprintln(out, `params := `+endpoint.ParamsStruct+`{}`)
			for _, field := range params[endpoint.ParamsStruct] {
				// generate validations
				fieldValue := getParamNameFromTags(&field)
				// lowercase or paramname if exists

				if field.Type == "int" {
					fmt.Fprintln(out, field.Name+`, err := strconv.Atoi(r.FormValue("`+fieldValue+`"))`)
					fmt.Fprintln(out, `if err != nil {`)
					fmt.Fprintln(out, `writeError(w, http.StatusBadRequest, "`+strings.ToLower(field.Name)+` must be int")`)
					fmt.Fprintln(out, `return`)
					fmt.Fprintln(out, `}`)
					fmt.Fprintln(out, `params.`+field.Name+`=`+field.Name)
				} else {
					fmt.Fprintln(out, `params.`+field.Name+`=r.FormValue("`+fieldValue+`")`)
				}

				generateValidations(out, &field)
			}

			// call function with error handling
			fmt.Fprintln(out, `ctx := r.Context()`)
			fmt.Fprintln(out, `result, err := h.`+endpoint.Name+`(ctx, params)`)
			handleErrorCode := `if err != nil {
				if specErr, ok := err.(ApiError); ok {
					writeError(w, specErr.HTTPStatus, specErr.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}`
			fmt.Fprintln(out, handleErrorCode)
			fmt.Fprintln(out, `resultData, err := json.Marshal(Response{Err: "", User: result})`)

			fmt.Fprintln(out, `if err != nil {`)
			fmt.Fprintln(out, ``)
			fmt.Fprintln(out, `}`)

			fmt.Fprintln(out, `w.Write(resultData)`)
			fmt.Fprintln(out, `}`)

			fmt.Fprintln(out) // empty line
		}
	}
}

func after(value string, a string) string {
	// Get substring after a string.
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}

func getParamNameFromTags(field *ParamMetaData) (paramValue string) {
	validations := strings.Split(field.Validations, ",")
	var fieldValue string
	for _, v := range validations {
		switch parsedVal := strings.Split(v, "="); parsedVal[0] {
		case "paramname":
			fieldValue = parsedVal[1]
		}
	}
	if fieldValue == "" {
		return strings.ToLower(field.Name)
	}
	return fieldValue
}

func generateValidations(out *os.File, field *ParamMetaData) {
	validations := strings.Split(field.Validations, ",")
	for _, v := range validations {
		switch parsedVal := strings.Split(v, "="); parsedVal[0] {
		case "required":
			fmt.Fprintln(out, `if params.`+field.Name+`==""{`)
			fmt.Fprintln(out, `writeError(w, http.StatusBadRequest, "login must me not empty")`)
			fmt.Fprintln(out, `return`)
			fmt.Fprintln(out, `}`)
		case "default":
			fmt.Fprintln(out, `if params.`+field.Name+`=="" {`)
			fmt.Fprintln(out, `params.`+field.Name+`="`+parsedVal[1]+`"`)
			fmt.Fprintln(out, `}`)
		case "enum":
			allowedValues := strings.Split(parsedVal[1], "|")

			var generateCondition func(cond string, i int) string
			generateCondition = func(cond string, i int) string {
				if i >= len(allowedValues) {
					return cond
				}
				var and string
				if i != len(allowedValues)-1 {
					and = " && "
				} else {
					and = ""
				}
				return generateCondition(cond+`params.`+field.Name+` != "`+allowedValues[i]+`"`+and, i+1)
			}
			cond := generateCondition(`if `, 0)
			// это проверяем в самом конце, чтобы проверка была после проставления значения по умолчанию
			defer fmt.Fprintln(out, cond+` {
				writeError(w, http.StatusBadRequest, "`+strings.ToLower(field.Name)+` must be one of [`+strings.Join(allowedValues, ", ")+`]")
				return
			}`)
		case "min":
			var minCheck string
			var errMsgPart string
			if field.Type == "int" {
				minCheck = `if params.` + field.Name
				errMsgPart = ` `
			} else {
				minCheck = `if len(params.` + field.Name + `)`
				errMsgPart = ` len `
			}
			fmt.Fprintln(out, minCheck+` < `+parsedVal[1]+`{
				writeError(w, http.StatusBadRequest, "`+strings.ToLower(field.Name)+errMsgPart+`must be >= `+parsedVal[1]+`")
				return
			}`)
		case "max":
			fmt.Fprintln(out, `if params.`+field.Name+` > `+parsedVal[1]+`{
				writeError(w, http.StatusBadRequest, "`+strings.ToLower(field.Name)+` must be <= `+parsedVal[1]+`")
				return
			}`)
		}
	}
}
