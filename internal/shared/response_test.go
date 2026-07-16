package shared

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestToJSONMap(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want int
	}{
		{"struct", struct{ Name string }{Name: "test"}, 1},
		{"map", map[string]string{"key": "val"}, 1},
		{"nil", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToJSONMap(tt.in)
			if tt.in == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil map")
			}
			if len(got) != tt.want {
				t.Errorf("expected %d keys, got %d", tt.want, len(got))
			}
		})
	}
}

func TestToJSONMap_Unmarshalable(t *testing.T) {
	ch := make(chan int)
	got := ToJSONMap(ch)
	if got != nil {
		t.Errorf("expected nil for unmarshalable type, got %v", got)
	}
}

func TestJSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	JSONSuccess(c, map[string]string{"foo": "bar"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data field, got %v", body)
	}
	if data["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", data["foo"])
	}
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	JSONError(c, http.StatusNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "not found" {
		t.Errorf("expected error message, got %v", body["error"])
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	InternalError(c, errors.New("something broke"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "internal server error" {
		t.Errorf("expected generic error, got %v", body["error"])
	}
}

func TestJSONPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	data := []string{"a", "b"}
	JSONPaginated(c, data, 10, 5, 0)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["total"] != float64(10) {
		t.Errorf("expected total=10, got %v", body["total"])
	}
	if body["limit"] != float64(5) {
		t.Errorf("expected limit=5, got %v", body["limit"])
	}
}
