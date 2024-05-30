package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddleware(t *testing.T) {
	// http.TimeoutHandler : http.Handler를 매개변수로 받아 http.Handler를 반환하는 미들웨어
	handler := http.TimeoutHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
			// 래핑된 http.Handler는 의도적으로 1분간 잠들어 클라이언트가 응답을 읽는 데 시간이 걸리는 것처럼 시뮬레이션해 http.Handler가 리턴되지 못하도록 함
			time.Sleep(time.Minute)
		}),
		time.Second,
		"Timed out while reading response",
	)

	r := httptest.NewRequest(http.MethodGet, "http://test/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// 핸들러가 1초 내에 응답되지 않으면, http.TimeoutHandler는 응답 상태 코드를 503 Service Unavailable로 설정
	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %q", resp.Status)
	}

	// response body 전체를 읽은 후
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	// response body를 닫고
	_ = resp.Body.Close()

	// response body에 올바르게 문자열을 썼는지 확인
	if actual := string(b); actual != "Timed out while reading response" {
		t.Logf("unexpected body: %q", actual)
	}
}
