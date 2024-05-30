package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/awoodbeck/gnp/ch09/handlers"
	"github.com/awoodbeck/gnp/ch09/middleware"
)

// HTTP/2 서버는 TLS 지원을 필요로 함
// 이를 위해 매개변수로 인증서의 경로와 인증서의 개인키 경로를 전달해 줘야 함
// 둘 중 하나의 값이 전달되지 않으면, 서버는 평문의 HTTP 연결을 대기함
var (
	addr  = flag.String("listen", "127.0.0.1:8080", "listen address")
	cert  = flag.String("cert", "", "certificate")
	pkey  = flag.String("key", "", "private key")
	files = flag.String("files", "./files", "static file directory")
)

func main() {
	flag.Parse()
	// run 함수에 CLI의 플래그 값을 전달
	err := run(*addr, *files, *cert, *pkey)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server stopped")
}

func run(addr, files, cert, pkey string) error {
	mux := http.NewServeMux()
	// 1. 정적 파일을 제공하기 위한 라우트
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			middleware.RestrictPrefix(
				".", http.FileServer(http.Dir("./files")),
			),
		),
	)
	// 2. 기본 라우트
	mux.Handle("/",
		handlers.Methods{
			http.MethodGet: http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					// http.ResponseWriter 인터페이스가 http.Pusher 객체인 경우, 별도의 요청 없이도 클라이언트에게 리소스를 푸시할 수 있음
					if pusher, ok := w.(http.Pusher); ok {
						// 서버 푸시를 할 때, 요청이 클라이언트에서 온 것으로 취급하므로 클라이언트 관점에서 리소스의 경로를 지정해 줌
						targets := []string{
							"/static/style.css",
							"/static/hiking.svg",
						}
						for _, target := range targets {
							if err := pusher.Push(target, nil); err != nil {
								log.Printf("%s push failed: %v", target, err)
							}
						}
					}
					// 리소스를 푸시해 준 뒤, 핸들러에서 응답을 처리
					// index.html 파일을 푸시해야 할 리소스보다 먼저 보낸 경우, 클라이언트의 브라우저에서는 푸시를 처리하기 전에 해당 리소스에 대한 요청을 보낼 수도 있음
					http.ServeFile(w, r, filepath.Join(files, "index.html"))
				},
			),
		},
	)
	// 3. 절대 경로 /2 를 위한 라우트
	// 이 파일이 기본 라우트에서 참조하는 동일한 리소스를 참조할 경우, 클라이언트의 웹 브라우저는 /2를 렌더링하는 동안 먼저 기본 라우트로 가서 푸시된 리소스를 사용하도록 함
	mux.Handle("/2",
		handlers.Methods{
			http.MethodGet: http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, filepath.Join(files, "index2.html"))
				},
			),
		},
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		for {
			// 서버가 os.Interrupt 시그널을 받으면, 서버를 닫음
			if <-c == os.Interrupt {
				_ = srv.Close()
				return
			}
		}
	}()

	log.Printf("Serving files in %q over %s\n", files, srv.Addr)

	var err error
	if cert != "" && pkey != "" {
		log.Println("TLS enabled")
		// 인증서의 경로와 개인키 경로를 모두 전달해주면, 서버는 ListenAndServeTLS 메서드를 사용해 TLS를 지원함
		// 인증서나 인증서의 개인키를 찾지 못하거나, 올바른 형식이 아니어서 파싱을 못할 경우, 메서드는 에러를 반환
		err = srv.ListenAndServeTLS(cert, pkey)
	} else {
		// 두 경로를 전달해 주지 않으면, 서버는 ListenAndServe 메서드를 사용
		err = srv.ListenAndServe()
	}

	if err == http.ErrServerClosed {
		err = nil
	}

	return err
}
