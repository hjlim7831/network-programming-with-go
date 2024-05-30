package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerWriteHeader(t *testing.T) {

	// 응답 상태 코드로 400 Bad Request를 생성한 뒤, response body로 문자열 Bad Request를 보내는 것 처럼 보이지만, 실제로 그렇게 동작하지 않음
	// ResponseWriter의 Write 메서드를 호출하면 Go는 암묵적으로 http.StatusOK 상태코드로 응답의 WriteHeader 메서드를 호출함
	// 응답의 상태 코드를 명시적이건 암묵적이건 WriteHeader 메서드 호출로 설정하고 나면, 변경 불가
	// 설계 이유 : 성공적이지 않은 경우에 대해서만 WriteHeader를 호출한다고 생각함
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
		// 위에서 이미 200 OK로 설정되었기 때문에, 이는 동작하지 않음
		w.WriteHeader(http.StatusBadRequest)
	}
	r := httptest.NewRequest(http.MethodGet, "http://test", nil)
	w := httptest.NewRecorder()
	handler(w, r)
	t.Logf("Response status: %q", w.Result().Status)

	handler = func(w http.ResponseWriter, r *http.Request) {
		// 여기서는 400 Bad Request 상태코드가 잘 설정됨
		// 이 코드는 http.Error 함수를 이용해 한 줄로 변경 가능
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("test"))
		// http.Error(w, "Bad Request", http.StatusBadRequest)
		// 위 함수는 Content-Type을 text/plain으로 설정하고, 상태 코드를 400 Bad Request로 설정하며, request body로 에러 메시지를 씀
	}
	r = httptest.NewRequest(http.MethodGet, "http://test", nil)
	w = httptest.NewRecorder()
	handler(w, r)
	t.Logf("Response status: %q", w.Result().Status)
}
