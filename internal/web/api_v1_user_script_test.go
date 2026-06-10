package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestAPIV1GetUserScript_ReturnsShape(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/user/script", nil)
	rec := httptest.NewRecorder()

	apiV1GetUserScript(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result APIResponse[map[string]string]
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("json.Decode: %v", err)
	}
	if !result.Success {
		t.Fatalf("success = false, error = %q", result.Error)
	}
	if _, ok := result.Data["script"]; !ok {
		t.Fatal("response missing 'script' key")
	}
	lang := result.Data["lang"]
	if lang != "js" && lang != "lua" {
		t.Fatalf("lang = %q, want js or lua", lang)
	}
}

func TestAPIV1PutUserScript_MalformedBody(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	req := httptest.NewRequest(http.MethodPut, "/admin/api/v1/user/script", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()

	apiV1PutUserScript(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
