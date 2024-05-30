package handlers

import (
	"html/template"
	"io"
	"net/http"
)

// http/template 패키지를 사용한 템플릿은 HTML에서 사용하는 문자열을 생성하거나, 결과로 응답 writer를 쓸 때 자동으로 이스케이핑함
// 클라이언트의 브라우저는 HTML 이스케이핑의 결과로, 응답 body의 결과를 HTML이나 자바스크립트로 해석하지 않고, 그냥 문자열로 인식할 수 있음
var t = template.Must(template.New("hello").Parse("Hello, {{.}}!"))

// 매개변수로 받은 함수를 http.HandlerFunc 타입으로 변환
// http.HandlerFunc : http.Handler 인터페이스를 구현함
// func(w http.ResponseWriter, r *http.Request)의 형태를 가진 함수를 http.HandlerFunc 타입으로 변환해 함수가 http.Handler 인터페이스를 구현하도록 함
func DefaultHandler() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			// defer로 호출된 함수가 request body를 소비하고 닫음
			// 클라이언트가 응답 body를 소비하고 TCP 연결을 닫아 TCP 세션을 재사용하는 것이 중요한 것처럼, 서버에서도 request body를 소비하고 TCP 연결을 닫는 것이 매우 중요
			// Go의 HTTP 클라이언트 : request body를 닫으면 암묵적으로 소비, 서버에서의 response body는 소비되지 않음
			// 확실하게 TCP 세션을 재사용하려면, 최소한 request body는 반드시 소비해야 함
			// TCP 연결은 선택적으로 닫아주기
			defer func(r io.ReadCloser) {
				_, _ = io.Copy(io.Discard, r)
				_ = r.Close()
			}(r.Body)

			var b []byte
			// 핸들러는 요청 메서드에 따라 다르게 응답
			switch r.Method {
			// 1. 클라이언트가 GET 요청을 하면
			case http.MethodGet:
				//  Hello, friend 라는 문자열을 응답 writer로 씀
				b = []byte("friend")
			// 2. 클라이언트가 POST 요청을 하면
			case http.MethodPost:
				var err error
				// 전체 request body를 읽음
				// 읽는 동안 에러 발생 시, Internal Server Error 메시지 작성 + 응답 상태 코드를 500으로 설정
				b, err = io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Internal server error",
						http.StatusInternalServerError)
					return
				}
				// 제대로 읽으면, Hello, <request body>를 그대로 넘김
			// 3. 그 외의 요청이 오면
			default:
				// 405 Method Not Allowed 상태를 응답함
				// RFC 표준을 준수하는 405 응답은 응답 헤더의 Allow 필드에 해당 핸들러가 처리하는 메서드가 무엇인지 알려줘야 함
				// 여기서는 RFC 표준을 준수하지 않음
				http.Error(w, "Method not allowed",
					http.StatusMethodNotAllowed)
				return
			}
			// 응답 body 쓰기
			_ = t.Execute(w, string(b))
		},
	)
}
