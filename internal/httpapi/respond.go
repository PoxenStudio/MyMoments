package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorBody 是稳定错误信封。
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 包含客户端可映射的错误码与字段原因。
type ErrorDetail struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

// FieldError 描述单个字段的校验失败。
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// APIError 携带 HTTP 状态与稳定错误码。
type APIError struct {
	Status  int
	Code    string
	Message string
	Fields  []FieldError
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

func badRequest(msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: "validation.invalid_field", Message: msg}
}

func unauthenticated(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: "auth.unauthenticated", Message: msg}
}

func forbidden(msg string) *APIError {
	return &APIError{Status: http.StatusForbidden, Code: "permission.denied", Message: msg}
}

func notFound(msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

func conflict(msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: "conflict", Message: msg}
}

func serverError(err error) *APIError {
	return &APIError{Status: http.StatusInternalServerError, Code: "server.unavailable", Message: "服务暂时不可用，请稍后再试"}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = serverError(err)
	}
	writeJSON(w, apiErr.Status, ErrorBody{Error: ErrorDetail{
		Code:    apiErr.Code,
		Message: apiErr.Message,
		Fields:  apiErr.Fields,
	}})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return badRequest("请求体格式不正确")
	}
	return nil
}
