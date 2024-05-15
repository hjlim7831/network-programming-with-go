package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/awoodbeck/gnp/ch06/tftp"
)

var (
	address = flag.String("a", "127.0.0.1:69", "listen address")
	payload = flag.String("p", "payload.svg", "file to serve to clients")
)

func main() {
	flag.Parse()
	// TFTP 서버가 바이트 슬라이스로 제공될 파일을 읽음
	p, err := ioutil.ReadFile(*payload)
	if err != nil {
		log.Fatal(err)
	}
	// 서버를 인스턴스화하고, 서버의 Payload 필드에 바이트 슬라이스를 할당함
	s := tftp.Server{Payload: p}
	// ListenAndServe 메서드를 호출해 요청을 수신할 UDP 연결을 설정
	// ListenAndServe 메서드는 내부적으로 연결 요청을 대기하는 서버의 Serve 메서드를 호출
	log.Fatal(s.ListenAndServe(*address))
}
