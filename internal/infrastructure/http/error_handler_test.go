package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

func TestWriteError_KnownErrorsMappedToExpectedStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "product not found", err: domain.ErrProductNotFound, want: http.StatusNotFound},
		{name: "missing ids typed error", err: &domain.MissingIDsError{Missing: []string{"p-404"}}, want: http.StatusNotFound},
		{name: "empty ids", err: domain.ErrEmptyIDs, want: http.StatusBadRequest},
		{name: "invalid field", err: domain.ErrInvalidField, want: http.StatusBadRequest},
		{name: "invalid pagination", err: domain.ErrInvalidPagination, want: http.StatusBadRequest},
		{name: "too many ids", err: domain.ErrTooManyIDs, want: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			writeError(c, tc.err)

			if w.Code != tc.want {
				t.Fatalf("expected status %d, got %d", tc.want, w.Code)
			}
		})
	}
}

func TestWriteError_UnknownErrorReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeError(c, errors.New("boom"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if len(c.Errors) == 0 {
		t.Fatalf("expected unknown error to be recorded in gin context")
	}
	if got := c.Errors[0].Err.Error(); got != "boom" {
		t.Fatalf("expected recorded error %q, got %q", "boom", got)
	}
	if c.Errors[0].Type != gin.ErrorTypePrivate {
		t.Fatalf("expected recorded error type %v, got %v", gin.ErrorTypePrivate, c.Errors[0].Type)
	}
}

func TestWriteError_UnknownErrorLogsCauseWithRequestIDAndKeepsGenericPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	r := gin.New()
	r.Use(RequestIDMiddleware(), LoggingMiddleware(logger))
	r.GET("/boom", func(c *gin.Context) {
		writeError(c, errors.New("db connection timeout"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set(requestIDHeader, "rid-500-xyz")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if got := body["error"]; got != "internal server error" {
		t.Fatalf("expected generic payload error, got %#v", got)
	}
	if _, has := body["db connection timeout"]; has {
		t.Fatalf("internal detail leaked in payload")
	}

	line := logs.String()
	if line == "" {
		t.Fatalf("expected structured logs")
	}
	if !bytes.Contains([]byte(line), []byte("\"request_id\":\"rid-500-xyz\"")) {
		t.Fatalf("expected request_id in logs, got %s", line)
	}
	if !bytes.Contains([]byte(line), []byte("db connection timeout")) {
		t.Fatalf("expected internal cause in logs, got %s", line)
	}
}
