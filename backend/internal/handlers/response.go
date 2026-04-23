package handlers

import (
	"encoding/json"
	"net/http"
)

type errorDetail struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Field   *string `json:"field,omitempty"`
}

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string, field string) {
	detail := errorDetail{Code: code, Message: message}
	if field != "" {
		detail.Field = &field
	}
	writeJSON(w, status, errorEnvelope{Error: detail})
}
