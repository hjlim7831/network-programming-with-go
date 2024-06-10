package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var addr = flag.String("listen", "localhost:8080", "listen address")

func main() {
	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	err := run(*addr, c)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server stopped")
}

func run(addr string, c chan os.Signal) error {
	mux := http.NewServeMux()
	mux.Handle("/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Caddy는 어느 클라이언트로부터 요청이 왔는지, 각 요청마다 클라이언트의 IP 주소를 X-Forwarded-For 헤더 필드 값에 추가함
			// 백엔드 서비스에서 이 IP 주소를 사용해 어느 클라이언트가 요청을 보냈는지 분간할 수 있음
			// ex) IP 주소를 기반으로 요청 거부도 가능
			clientAddr := r.Header.Get("X-Forwarded-For")
			log.Printf("%s -> %s -> %s", clientAddr, r.RemoteAddr, r.URL)
			_, _ = w.Write(index)
		}),
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
	}

	go func() {
		for {
			if <-c == os.Interrupt {
				_ = srv.Close()
				return
			}
		}
	}()

	fmt.Printf("Listening on %s ...\n", srv.Addr)
	err := srv.ListenAndServe()

	if err == http.ErrServerClosed {
		err = nil
	}

	return err
}

var index = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Caddy Backend Test</title>
    <link href="/style.css" rel="stylesheet">
</head>
<body>
    <p><img src="/hiking.svg" alt="hiking gopher"></p>
</body>
</html>`)
