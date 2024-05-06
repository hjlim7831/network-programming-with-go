package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

// 리눅스상의 ping 커맨드가 제공하는 기능의 일부를 흉내 낼 수 있는 몇몇 커맨드 라인 옵션을 정의 (명령줄로 입력하는 인수 정의)
var (
	count    = flag.Int("c", 3, "number of pings: <= 0 means forever")
	interval = flag.Duration("i", time.Second, "interval between pings")
	timeout  = flag.Duration("W", 5*time.Second, "time to wait for a reply")
)

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] host:port\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Print("host:port is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	target := flag.Arg(0)
	fmt.Println("PING", target)

	if *count <= 0 {
		fmt.Println("CTRL+C to stop.")
	}

	msg := 0

	for (*count <= 0) || (msg < *count) {
		msg++
		fmt.Print(msg, " ")

		start := time.Now()
		// 원격 호스트의 TCP 포트로 연결 수립을 시도
		// 원격 호스트가 응답하지 않을 경우를 대비해 적절한 타임아웃 시간을 설정
		c, err := net.DialTimeout("tcp", target, *timeout)
		// TCP 핸드셰이크를 마치는 데에 걸리는 시간을 추적
		// 이 시간을 출발지 호스트와 원격 호스트 간에 ping이 도달하는 시간으로 생각하면 됨
		dur := time.Since(start)

		if err != nil {
			fmt.Printf("fail in %s: %v\n", dur, err)
			// 일시적인 에러가 아닌 경우
			if nErr, ok := err.(net.Error); !ok || !nErr.Temporary() {
				// 종료
				os.Exit(1)
			}
			// 일시적인 에러라면? 재시도 수행 (다음 for문으로 넘어감)
		} else {
			_ = c.Close()
			fmt.Println(dur)
		}

		time.Sleep(*interval)
	}
}
