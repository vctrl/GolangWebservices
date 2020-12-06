package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Response struct {
	Err  string      `json:"error"`
	User interface{} `json:"response,omitempty"`
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	result, err := json.Marshal(Response{Err: message})
	if err != nil {

	}
	w.WriteHeader(statusCode)
	w.Write(result)
}

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		h.handleProfile(w, r)
	case "/user/create":
		h.handleCreate(w, r)
	default:
		writeError(w, http.StatusNotFound, "unknown method")
		return
	}
}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		h.handleCreate(w, r)
	default:
		writeError(w, http.StatusNotFound, "unknown method")
		return
	}
}

func (h *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {
	params := ProfileParams{}
	params.Login = r.FormValue("login")
	if params.Login == "" {
		writeError(w, http.StatusBadRequest, "login must me not empty")
		return
	}
	ctx := r.Context()
	result, err := h.Profile(ctx, params)
	if err != nil {
		if specErr, ok := err.(ApiError); ok {
			writeError(w, specErr.HTTPStatus, specErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resultData, err := json.Marshal(Response{Err: "", User: result})
	if err != nil {

	}
	w.Write(resultData)
}

func (h *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		writeError(w, http.StatusForbidden, "unauthorized")
		return
	}
	if r.Method != "POST" {
		writeError(w, http.StatusNotAcceptable, "bad method")
		return
	}

	params := CreateParams{}
	params.Login = r.FormValue("login")
	if params.Login == "" {
		writeError(w, http.StatusBadRequest, "login must me not empty")
		return
	}
	if len(params.Login) < 10 {
		writeError(w, http.StatusBadRequest, "login len must be >= 10")
		return
	}
	params.Name = r.FormValue("full_name")
	params.Status = r.FormValue("status")
	if params.Status == "" {
		params.Status = "user"
	}
	if params.Status != "user" && params.Status != "moderator" && params.Status != "admin" {
		writeError(w, http.StatusBadRequest, "status must be one of [user, moderator, admin]")
		return
	}
	Age, err := strconv.Atoi(r.FormValue("age"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "age must be int")
		return
	}
	params.Age = Age
	if params.Age < 0 {
		writeError(w, http.StatusBadRequest, "age must be >= 0")
		return
	}
	if params.Age > 128 {
		writeError(w, http.StatusBadRequest, "age must be <= 128")
		return
	}
	ctx := r.Context()
	result, err := h.Create(ctx, params)
	if err != nil {
		if specErr, ok := err.(ApiError); ok {
			writeError(w, specErr.HTTPStatus, specErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resultData, err := json.Marshal(Response{Err: "", User: result})
	if err != nil {

	}
	w.Write(resultData)
}

func (h *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Auth") != "100500" {
		writeError(w, http.StatusForbidden, "unauthorized")
		return
	}
	if r.Method != "POST" {
		writeError(w, http.StatusNotAcceptable, "bad method")
		return
	}

	params := OtherCreateParams{}
	params.Username = r.FormValue("username")
	if params.Username == "" {
		writeError(w, http.StatusBadRequest, "login must me not empty")
		return
	}
	if len(params.Username) < 3 {
		writeError(w, http.StatusBadRequest, "username len must be >= 3")
		return
	}
	params.Name = r.FormValue("account_name")
	params.Class = r.FormValue("class")
	if params.Class == "" {
		params.Class = "warrior"
	}
	if params.Class != "warrior" && params.Class != "sorcerer" && params.Class != "rouge" {
		writeError(w, http.StatusBadRequest, "class must be one of [warrior, sorcerer, rouge]")
		return
	}
	Level, err := strconv.Atoi(r.FormValue("level"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "level must be int")
		return
	}
	params.Level = Level
	if params.Level < 1 {
		writeError(w, http.StatusBadRequest, "level must be >= 1")
		return
	}
	if params.Level > 50 {
		writeError(w, http.StatusBadRequest, "level must be <= 50")
		return
	}
	ctx := r.Context()
	result, err := h.Create(ctx, params)
	if err != nil {
		if specErr, ok := err.(ApiError); ok {
			writeError(w, specErr.HTTPStatus, specErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resultData, err := json.Marshal(Response{Err: "", User: result})
	if err != nil {

	}
	w.Write(resultData)
}
