package main

import (
	"io"
	"log"
	"net"
	"os"
)

// Monitor 구조체 : 네트워크 트래픽을 로깅하기 위한 log.Logger를 임베딩
// 서버의 네트워크 트래픽을 로깅하기 위한 목적
type Monitor struct {
	*log.Logger
}

// Write 메서드는 io.Writer 인터페이스를 구현
// io.TeeReader와 io.MultiWriter 함수가 io.Writer 인터페이스를 매개변수로 받기 때문
func (m *Monitor) Write(p []byte) (int, error) {
	return len(p), m.Output(2, string(p))
}

func ExampleMonitor() {
	// os.Stdout(표준 출력)으로 데이터를 쓰는 Monitor 구조체의 인스턴스 생성
	monitor := &Monitor{Logger: log.New(os.Stdout, "monitor: ", 0)}

	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		monitor.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		b := make([]byte, 1024)
		// io.TeeReader 함수에서는 monitor 인스턴스 변수와 함께 연결 객체를 사용
		// io.Reader는 네트워크 연결로부터 데이터를 읽고, 읽은 데이터를 모니터에 출력한 후 함수를 호출한 호출자에게 전달
		r := io.TeeReader(conn, monitor)
		n, err := r.Read(b)
		if err != nil && err != io.EOF {
			monitor.Println(err)
			return
		}
		// 서버의 출력 결과를 생성한 io.MultiWriter를 이용해 네트워크 연결과 모니터에 로깅
		w := io.MultiWriter(conn, monitor)
		_, err = w.Write(b[:n]) // echo the message
		if err != nil && err != io.EOF {
			monitor.Println(err)
			return
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		monitor.Fatal(err)
	}
	// Test\n 라는 메시지를 전송하면 os.Stdout에 총 두 번 로깅됨
	// 1) 네트워크 연결로부터 데이터를 읽을 때
	// 2) 클라이언트에게 메시지를 되돌려줄 때
	_, err = conn.Write([]byte("Test\n"))
	if err != nil {
		monitor.Fatal(err)
	}

	_ = conn.Close()
	<-done

	// Output:
	// monitor: Test
	// monitor: Test
}
