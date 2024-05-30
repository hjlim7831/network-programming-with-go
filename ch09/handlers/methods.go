package handlers

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strings"
)

// Methods라는 새로운 타입을 생성
// 키 이름 : HTTP 메서드, 값 : http.Handler를 갖는 맵
type Methods map[string]http.Handler

// Methods 타입에는 ServeHTTP 메서드가 존재
// http.Handler 인터페이스가 구현되었으므로, Methods 자체를 핸들러로 사용 가능
// Methods 타입은 요청을 보고 적절한 핸들러로 라우팅하므로, 멀티플렉서임
func (h Methods) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// defer로 request body를 소비하고 연결을 닫는 함수 호출
	// 맵의 값으로 존재하는 핸들러에서 직접 해주지 않아도 됨
	defer func(r io.ReadCloser) {
		_, _ = io.Copy(io.Discard, r)
		_ = r.Close()
	}(r.Body)

	// 요청 메서드(r.Method)를 보고 맵에서 요청 메서드에 해당하는 핸들러(handler)를 가져옴
	if handler, ok := h[r.Method]; ok {
		// 혹시 모르게 발생할 패닉을 방지하기 위해, ServeHTTP 메서드는 요청 메서드에 해당하는 핸들러가 nil이 아닌지 확인
		// 만약 nil이면, 500 Internal Server Error 반환
		if handler == nil {
			http.Error(w, "Internal server error",
				http.StatusInternalServerError)
		} else {
			// 정상적으로 핸들러를 가져오면, 핸들러의 ServeHTTP 메서드를 호출
			handler.ServeHTTP(w, r)
		}

		return
	}
	// 요청 메서드에 해당하는 키가 맵에 없는 경우
	// 현재 맵에서 지원하는 메서드의 목록을 Allow 헤더에 반환
	w.Header().Add("Allow", h.allowedMethods())
	// 클라이언트가 명시적으로 OPTIONS 메서드를 요청했는지 확인
	// OPTIONS 메서드의 요청을 받으면 ServeHTTP 메서드는 클라이언트에게 200 OK를 응답
	if r.Method != http.MethodOptions {
		// 이도저도 아닌 경우, 405 Method Not Allowed를 응답
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h Methods) allowedMethods() string {
	a := make([]string, 0, len(h))

	for k := range h {
		a = append(a, k)
	}
	sort.Strings(a)

	return strings.Join(a, ", ")
}

// GET, POST, OPTIONS 메서드를 지원
// 이 함수가 반환하는 핸들러는 그대로 handlers.DefaultHandler 함수에서 반환하는 핸들러로 교체 가능
func DefaultMethodsHandler() http.Handler {
	return Methods{
		// GET 메서드 : Hello, friend! 라는 메시지를 response body로 씀
		http.MethodGet: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("Hello, friend!"))
			},
		),
		// POST 메서드 : response body로 전송한 내용을 이스케이핑해, Hello, <POST response body>! 를 전송
		http.MethodPost: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Internal server error",
						http.StatusInternalServerError)
					return
				}

				_, _ = fmt.Fprintf(w, "Hello, %s!",
					html.EscapeString(string(b)))
			},
		),
	}
}
