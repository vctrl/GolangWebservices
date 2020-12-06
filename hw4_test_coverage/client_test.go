package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type TestCase struct {
	Request *SearchRequest
	Result  *SearchResponse
	IsError bool
}

type UserData struct {
	ID        int    `xml:"id"`
	FirstName string `xml:"first_name" json:"-"`
	LastName  string `xml:"last_name" json:"-"`
	Name      string `xml:"-"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type Root struct {
	Users []UserData `xml:"row"`
}

func SearchServerStatusUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}

func SearchServerStatusInternalServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func SearchServerStatusBadRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func SearchServerStatusBadRequestUnknownError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	resultErrorData, err := json.Marshal(SearchErrorResponse{Error: "UnknownBadRequestError"})
	if err != nil {
		log.Fatal(err)
	}
	w.Write(resultErrorData)
}

func SearchServerBrokenJsonError(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("{\"hello: 1}"))
}

func SearchServerTimeoutError(w http.ResponseWriter, r *http.Request) {
	time.Sleep(client.Timeout + time.Millisecond)
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		log.Fatal(err)
	}
	root := Root{}
	if err := xml.Unmarshal(data, &root); err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(root.Users); i++ {
		root.Users[i].Name = root.Users[i].FirstName + " " + root.Users[i].LastName
	}

	query := r.FormValue("query")
	users := filter(root.Users, query)
	orderField := r.FormValue("order_field")
	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		log.Fatal(err)
	}
	users, err = order(users, orderField, orderBy)
	if err != nil {
		fmt.Println("ya tut")
		resultErrorData, err := json.Marshal(SearchErrorResponse{Error: "ErrorBadOrderField"})
		if err != nil {
			log.Fatal(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resultErrorData)
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		log.Fatal(err)
	}
	if limit < len(users) {
		users = users[:limit]
	}

	resultData, err := json.Marshal(users)
	if err != nil {
		log.Fatal(err)
	}

	w.Write(resultData)
}

func filter(users []UserData, query string) []UserData {
	if query == "" {
		return users
	}
	result := make([]UserData, 0, 5)
	for _, user := range users {
		if strings.Contains(user.Name, query) || strings.Contains(user.About, query) {
			result = append(result, user)
		}
	}
	return result
}

func order(users []UserData, field string, orderBy int) ([]UserData, error) {
	if field == "" {
		field = "Name"
	}
	if field != "Id" && field != "Age" && field != "Name" { //todo array on the top orderAllowedFields
		return nil, errors.New("Unknown order field")
	}
	if orderBy == OrderByAsIs {
		return users, nil
	}
	sort.Slice(users, func(i, j int) bool {
		val1 := reflect.ValueOf(users[i]).FieldByName(field).Interface().(string)
		val2 := reflect.ValueOf(users[j]).FieldByName(field).Interface().(string)

		if orderBy == OrderByAsc {
			return val1 < val2
		}
		return val1 > val2
	})
	return users, nil
}

func TestNextPageIsTrueHappyCase(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{
			Limit:      5,
			Offset:     1,
			OrderBy:    OrderByAsc,
			OrderField: "",
			Query:      "",
		},
		Result: &SearchResponse{
			Users:    []User{User{Id: 15, Name: "Allison Valdez", Age: 21, About: "Labore excepteur voluptate velit occaecat est nisi minim. Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. Ullamco sint nisi voluptate cillum nostrud aliquip et minim. Enim duis esse do aute qui officia ipsum ut occaecat deserunt. Pariatur pariatur nisi do ad dolore reprehenderit et et enim esse dolor qui. Excepteur ullamco adipisicing qui adipisicing tempor minim aliquip.\n", Gender: "male"}, User{Id: 16, Name: "Annie Osborn", Age: 35, About: "Consequat fugiat veniam commodo nisi nostrud culpa pariatur. Aliquip velit adipisicing dolor et nostrud. Eu nostrud officia velit eiusmod ullamco duis eiusmod ad non do quis.\n", Gender: "female"}, User{Id: 19, Name: "Bell Bauer", Age: 26, About: "Nulla voluptate nostrud nostrud do ut tempor et quis non aliqua cillum in duis. Sit ipsum sit ut non proident exercitation. Quis consequat laboris deserunt adipisicing eiusmod non cillum magna.\n", Gender: "male"}, User{Id: 22, Name: "Beth Wynn", Age: 31, About: "Proident non nisi dolore id non. Aliquip ex anim cupidatat dolore amet veniam tempor non adipisicing. Aliqua adipisicing eu esse quis reprehenderit est irure cillum duis dolor ex. Laborum do aute commodo amet. Fugiat aute in excepteur ut aliqua sint fugiat do nostrud voluptate duis do deserunt. Elit esse ipsum duis ipsum.\n", Gender: "female"}, User{Id: 5, Name: "Beulah Stark", Age: 30, About: "Enim cillum eu cillum velit labore. In sint esse nulla occaecat voluptate pariatur aliqua aliqua non officia nulla aliqua. Fugiat nostrud irure officia minim cupidatat laborum ad incididunt dolore. Fugiat nostrud eiusmod ex ea nulla commodo. Reprehenderit sint qui anim non ad id adipisicing qui officia Lorem.\n", Gender: "female"}},
			NextPage: true,
		},
		IsError: false,
	}

	RunTestCase(t, &testCase, SearchServer)
}

func TestLimitCannotBeNegative(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{
			Limit:      -1,
			Offset:     1,
			OrderBy:    OrderByAsc,
			OrderField: "",
			Query:      "",
		},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServer)
}

func TestLimitMaxValueIs25(t *testing.T) {
	req := &SearchRequest{
		Limit:      100,
		Offset:     1,
		OrderBy:    OrderByAsc,
		OrderField: "",
		Query:      "",
	}
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	s := &SearchClient{
		URL: ts.URL,
	}
	result, err := s.FindUsers(*req)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
	resultLength := len((*result).Users)
	if resultLength != 25 {
		t.Errorf("wrong result length, expected %d, got %d", 25, resultLength)
	}
}

func TestOffsetCannotBeNegative(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{
			Limit:      1,
			Offset:     -1,
			OrderBy:    OrderByAsc,
			OrderField: "",
			Query:      "",
		},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServer)
}

func TestNextPageIsFalseHappyCase(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{
			Limit:      10,
			Offset:     0,
			OrderBy:    OrderByAsc,
			OrderField: "",
			Query:      "Cruz",
		},
		Result: &SearchResponse{
			Users:    []User{User{Id: 12, Name: "Cruz Guerrero", Age: 36, About: "Sunt enim ad fugiat minim id esse proident laborum magna magna. Velit anim aliqua nulla laborum consequat veniam reprehenderit enim fugiat ipsum mollit nisi. Nisi do reprehenderit aute sint sit culpa id Lorem proident id tempor. Irure ut ipsum sit non quis aliqua in voluptate magna. Ipsum non aliquip quis incididunt incididunt aute sint. Minim dolor in mollit aute duis consectetur.\n", Gender: "male"}},
			NextPage: false,
		},
		IsError: false,
	}

	RunTestCase(t, &testCase, SearchServer)
}

func TestFailRequestByTimeout(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServerTimeoutError)
}

func TestFailRequestByUnknownError(t *testing.T) {
	s := &SearchClient{
		URL: "",
	}

	tc := TestCase{Result: nil,
		Request: &SearchRequest{},
		IsError: true}
	result, err := s.FindUsers(*tc.Request)
	CheckResult(t, result, err, &tc)
}

func TestServerReturnsUnathorized(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServerStatusUnauthorized)
}

func TestServerReturnsInternalServerError(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServerStatusInternalServerError)
}

func TestServerReturnsBadRequest(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServerStatusBadRequest)
}

func TestServerReturnsBrokenJson(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServerBrokenJsonError)
}

func TestServerReturnsUnknownBadRequestError(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServerStatusBadRequestUnknownError)
}

func TestUnknownOrderField(t *testing.T) {
	testCase := TestCase{
		Request: &SearchRequest{
			OrderField: "unknown",
		},
		Result:  nil,
		IsError: true,
	}

	RunTestCase(t, &testCase, SearchServer)
}

func RunTestCase(t *testing.T, testCase *TestCase, handleFunc func(http.ResponseWriter, *http.Request)) {
	ts := httptest.NewServer(http.HandlerFunc(handleFunc))
	s := &SearchClient{
		URL: ts.URL,
	}
	result, err := s.FindUsers(*testCase.Request)
	CheckResult(t, result, err, testCase)
}

func CheckResult(t *testing.T, result *SearchResponse, err error, testCase *TestCase) {
	if err != nil && !testCase.IsError {
		t.Errorf("unexpected error: %#v", err)
	}
	if err == nil && testCase.IsError {
		t.Errorf("expected error, got nil")
	}
	if !reflect.DeepEqual(result, testCase.Result) {
		t.Errorf("wrong result, expected %#v, got %#v", result, testCase.Result)
	}
}
