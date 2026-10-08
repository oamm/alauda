package problem

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Details struct {
	Type   string              `json:"type"`
	Title  string              `json:"title"`
	Status int                 `json:"status"`
	Code   string              `json:"code"`
	Detail string              `json:"detail"`
	Errors map[string][]string `json:"errors,omitempty"`
}

func Write(w http.ResponseWriter, status int, code, detail string, fields map[string][]string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Details{Type: "https://alauda.dev/problems/" + strings.ReplaceAll(code, "_", "-"), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, Errors: fields})
}
