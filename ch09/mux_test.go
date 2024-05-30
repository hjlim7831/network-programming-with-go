package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 미들웨어를 사용해 request body를 소비하고 닫음
// 이 미들웨어에서는 먼저 다음에 처리될 핸들러를 호출하고, request body를 소비한 뒤 닫아 버림
// 이미 소비되고 닫힌 request body를 또 다시 닫는다고 해서 문제될 것은 전혀 없음
func drainAndClose(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
		},
	)
}

func TestSimpleMux(t *testing.T) {
	serveMux := http.NewServeMux()
	// 새로운 멀티플렉서를 생성하고, 멀티플렉서의 HandleFunc 메서드를 이용해 세 개의 라우트를 등록
	// 1. 기본 경로 혹은 공백 URL 경로(/) : 204 No Content 상태
	// 이 라우트는 등록된 다른 라우트가 모두 일치하지 않는 경우에 실행됨
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	// 2. 응답에 문자열 "Hello friend"를 쓰는 /hello
	// trailing slash 존재 X
	// 해당 경로가 절대 경로임을 의미함
	// -> 정확하게 일치해야 하는 것으로 취급
	serveMux.HandleFunc("/hello", func(w http.ResponseWriter,
		r *http.Request) {
		_, _ = fmt.Fprint(w, "Hello friend.")
	})
	// 3. 응답에 문자열 "Why, hello there."을 쓰는 /hello/there/
	// trailing slash가 존재함
	// 라우트에 트레일링 슬래시가 존재하는 것은, 해당 경로가 서브트리임을 의미함
	// -> "접두사"가 일치해야 하는 것으로 취급
	// ex) /hello/there/you 도 접두사가 일치한다고 봄

	// Go의 멀티플렉서는 트레일링 슬래시를 갖지 않는 URL 경로를 리다이렉트할 수도 있음
	// 그런 경우, http.ServeMux 핸들러는 먼저 일치하는 절대 경로를 찾도록 시도함
	// ex) /hello/there -> /hello/there/ 로 만든 후 핸들러로 넘겨, 클라이언트가 이에 대한 응답을 받도록 함.
	// 새로운 경로로 영구 리다이렉트가 됨
	serveMux.HandleFunc("/hello/there/", func(w http.ResponseWriter,
		r *http.Request) {
		_, _ = fmt.Fprint(w, "Why, hello there.")
	})
	mux := drainAndClose(serveMux)

	testCases := []struct {
		path     string
		response string
		code     int
	}{
		// 위의 3개는 멀티플렉서에 등록된 패턴과 정확히 일치
		{"http://test/", "", http.StatusNoContent},
		{"http://test/hello", "Hello friend.", http.StatusOK},
		{"http://test/hello/there/", "Why, hello there.", http.StatusOK},
		// 정확한 경로와 일치하지는 않음
		// 트레일링 슬래시를 더하고 나서야 정확히 일치하는 패턴을 발견함
		// -> 301 Moved Permanently 상태를 응답함. request body에 새로운 경로에 대한 링크를 포함함
		{"http://test/hello/there",
			"<a href=\"/hello/there/\">Moved Permanently</a>.\n\n",
			http.StatusMovedPermanently},
		// /hello/there/의 서브트리에 매칭되어, "Why, hello there." 응답을 받게 됨
		{"http://test/hello/there/you", "Why, hello there.", http.StatusOK},
		// 아래 두 개는 기본 경로인 /에 매칭되어, 204 No Content 상태를 응답함
		{"http://test/something/else/entirely", "", http.StatusNoContent},
		{"http://test/hello/you", "", http.StatusNoContent},
	}

	for i, c := range testCases {
		r := httptest.NewRequest(http.MethodGet, c.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		resp := w.Result()

		if actual := resp.StatusCode; c.code != actual {
			t.Errorf("%d: expected code %d; actual %d", i, c.code, actual)
		}

		// response body에 대해서도 소비하고
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		// 닫아주기
		_ = resp.Body.Close()

		if actual := string(b); c.response != actual {
			t.Errorf("%d: expected response %q; actual %q", i,
				c.response, actual)
		}
	}
}
