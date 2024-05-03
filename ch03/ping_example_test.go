package ch03

import (
	"context"
	"fmt"
	"io"
	"time"
)

// ping.go 함수를 사용해보는 코드
func ExamplePinger() {
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe() // net.Conn 대신 사용
	done := make(chan struct{})
	// Pinger의 타이머를 리셋하기 위해 사용하는 시그널로 버퍼 채널 생성
	resetTimer := make(chan time.Duration, 1)
	// 채널을 Pinger 함수로 넘기기 전에 resetTimer 채널에서 초기 핑 타이머 간격으로 1초를 설정
	// Pinger의 타이머를 초기화하고 핑 메시지를 writer에 쓸 때, 이 간격을 사용
	resetTimer <- time.Second // 초기 핑 간격

	go func() {
		Pinger(ctx, w, resetTimer)
		close(done)
	}()
	// 핑 타이머를 주어진 값(d)으로 초기화하고, 주어진 reader로부터 핑 메시지를 받을 때까지 대기
	receivePing := func(d time.Duration, r io.Reader) {
		if d >= 0 {
			fmt.Printf("resetting timer (%s)\n", d)
			resetTimer <- d
		}

		now := time.Now()
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("received %q (%s)\n",
			buf[:n], time.Since(now).Round(100*time.Millisecond))
	}
	// 일련의 시간 간격을 밀리초 단위로 정의해 만든 int64 배열을 for 루프에서 순회해 각각의 값을 receivePing 함수로 전달

	for i, v := range []int64{0, 200, 300, 0, -1, -1, -1} {
		// 핑 메시지 수신에 걸린 시간을 표준 출력으로 출력
		fmt.Printf("Run %d:\n", i+1)
		receivePing(time.Duration(v)*time.Millisecond, r)
	}

	cancel()
	<-done // 컨텍스트가 취소된 이후 pinger가 종료되었는지 확인

	// Output:
	// Run 1:
	// resetting timer (0s)
	// received "ping" (1s)
	// Run 2:
	// resetting timer (200ms)
	// received "ping" (200ms)
	// Run 3:
	// resetting timer (300ms)
	// received "ping" (300ms)
	// Run 4:
	// resetting timer (0s)
	// received "ping" (300ms)
	// Run 5:
	// received "ping" (300ms)
	// Run 6:
	// received "ping" (300ms)
	// Run 7:
	// received "ping" (300ms)

}
