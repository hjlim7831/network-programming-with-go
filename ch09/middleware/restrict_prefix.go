package middleware

import (
	"net/http"
	"path"
	"strings"
)

func RestrictPrefix(prefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// 요청 URL 경로를 검사. 주어진 접두사로 시작하는지 확인
			for _, p := range strings.Split(path.Clean(r.URL.Path), "/") {
				// 요청 URL 경로가 주어진 접두사로 시작하는 경우, 그대로 404 Not Found 상태를 응답함
				if strings.HasPrefix(p, prefix) {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
			}
			next.ServeHTTP(w, r)
		},
	)
}
