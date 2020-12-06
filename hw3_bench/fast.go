package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jlexer"
	"github.com/mailru/easyjson/jwriter"
)

var dataPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 64))
	},
}

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)
	seenBrowsers := make(map[string]bool)
	foundUsers := strings.Builder{}
	var i int
	for scanner.Scan() {
		var user User
		bytesBuffer := dataPool.Get().(*bytes.Buffer)
		bytesBuffer.WriteString(scanner.Text())
		err := user.UnmarshalJSON(bytesBuffer.Bytes())
		bytesBuffer.Reset()
		dataPool.Put(bytesBuffer)
		if err != nil {
			panic(err)
		}
		if filterUser(user, seenBrowsers) {
			email := strings.ReplaceAll(user.Email, "@", " [at] ")
			foundUsers.WriteString(fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email))
		}
		i++
	}
	fmt.Fprintln(out, "found users:\n"+foundUsers.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}

func filterUser(user User, seenBrowsers map[string]bool) bool {
	isAndroid := false
	isMSIE := false

	browsers := user.Browsers

	for _, browser := range browsers {
		if strings.Contains(browser, "Android") {
			isAndroid = true
		}
		if strings.Contains(browser, "MSIE") {
			isMSIE = true
		}
		if strings.Contains(browser, "Android") || strings.Contains(browser, "MSIE") {
			if !seenBrowsers[browser] {
				seenBrowsers[browser] = true
			}
		}
	}

	if !(isAndroid && isMSIE) {
		return false
	}

	return true
	// log.Println("Android and MSIE user:", user["name"], user["email"])
}

type User struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Browsers []string `json:"browsers"`
}

// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson6a975c40DecodeUsersViktorGoWorkspaceSrc(in *jlexer.Lexer, out *User) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "name":
			out.Name = string(in.String())
		case "email":
			out.Email = string(in.String())
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson6a975c40EncodeUsersViktorGoWorkspaceSrc(out *jwriter.Writer, in User) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"name\":"
		out.RawString(prefix[1:])
		out.String(string(in.Name))
	}
	{
		const prefix string = ",\"email\":"
		out.RawString(prefix)
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"browsers\":"
		out.RawString(prefix)
		if in.Browsers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Browsers {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v User) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson6a975c40EncodeUsersViktorGoWorkspaceSrc(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v User) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson6a975c40EncodeUsersViktorGoWorkspaceSrc(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *User) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson6a975c40DecodeUsersViktorGoWorkspaceSrc(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *User) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson6a975c40DecodeUsersViktorGoWorkspaceSrc(l, v)
}
