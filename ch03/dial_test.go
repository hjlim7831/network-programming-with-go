package ch03

import (
	"io"
	"net"
	"testing"
)

func TestDial(t *testing.T) {
	// Create a listener on a random port.
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	// done 신호를 받는 채널 생성
	done := make(chan struct{})
	go func() {
		// 이 함수가 끝날 때, done 채널로 신호 보내기
		defer func() { done <- struct{}{} }()
		// 하나 이상의 연결을 처리하려면, for문으로 서버가 계속해서 수신 연결 요청을 수락하고 고루틴에서 해당 연결을 처리하고, 다시 for문으로 돌아와 다음 연결 요청을 수락할 수 있도록 대기해야 함
		// TCP 수신 연결을 루프에서 받아들이고, 각 연결 처리 로직을 담당하는 고루틴을 시작
		for {
			// 리스너가 수신 연결을 감지하고, 클라이언트와 서버 간의 TCP 핸드셰이크 절차가 완료될 때까지 블로킹됨
			conn, err := listener.Accept()
			if err != nil {
				t.Log(err)
				return
			}
			// 커넥션 핸들러 (handler)
			// 반드시 고루틴을 사용해 각 연결을 처리해야 함
			// 고루틴을 쓰지 않고 직렬화된 코드를 작성하는 것도 가능하지만, Go 언어의 강점을 활용하지 못하게 되어 상당히 비효율적임
			go func(c net.Conn) {
				defer func() {
					c.Close()
					done <- struct{}{}
				}()
				// 소켓으로부터 1024 바이트를 읽어 수신한 데이터를 로깅
				buf := make([]byte, 1024)
				for {
					// FIN 패킷을 받고 나면 Read 메서드는 io.EOF 에러를 반환
					n, err := c.Read(buf)
					if err != nil {
						if err != io.EOF {
							t.Error(err)
						}
						return
					}

					t.Logf("received: %q", buf[:n])
				}
			}(conn)
		}
	}()

	// 클라이언트 측. 연결 시도
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// 우아한 종료 시작
	conn.Close()     // conn.Close() 메서드를 호출한 뒤
	<-done           // 실제로 무사히 종료가 되었는지 done 메서드로 신호가 오는 것을 기다림
	listener.Close() // listener.Close() 메서드를 호출한 뒤
	<-done           // 실제로 listener가 무사히 종료되었는지 done 메서드로 신호가 오는 것을 기다림
}
