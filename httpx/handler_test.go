package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusHandler(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			res := httptest.NewRecorder()

			handler := StatusHandler(status)
			handler.ServeHTTP(res, req)

			if res.Code != status {
				t.Error(res.Code)
			}
		})
	}
}
