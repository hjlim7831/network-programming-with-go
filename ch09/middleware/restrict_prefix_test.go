package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRestrictPrefix(t *testing.T) {
	// 요청을 RestrictPrefix 미들웨어로 넘기기 전, 먼저 http.StripPrefix 미들웨어로 넘김
	// 그리고 RestrictPrefix 미들웨어에서 리소스 경로가 통과되면, 요청을 http.FileServer로 넘김
	// RestrictPrefix 미들웨어 : 요청의 리소스 경로를 평가해, 클라이언트가 현재 요청하는 파일이 파일의 존재 유무와 상관없이 민감한 파일인지를 검사
	// 민감한 파일이라면? 요청을 http.FileServer로 넘기지 않고, 그대로 클라이언트에게 에러를 응답
	handler := http.StripPrefix("/static/",
		RestrictPrefix(".", http.FileServer(http.Dir("../files/"))),
	)

	testCases := []struct {
		path string
		code int
	}{
		{"http://test/static/sage.svg", http.StatusOK},
		{"http://test/static/.secret", http.StatusNotFound},
		{"http://test/static/.dir/secret", http.StatusNotFound},
	}

	for i, c := range testCases {
		r := httptest.NewRequest(http.MethodGet, c.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		actual := w.Result().StatusCode
		if c.code != actual {
			t.Errorf("%d: expected %d; actual %d", i, c.code, actual)
		}
	}
}
