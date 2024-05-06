package send

import (
	"errors"
	"log"
	"net"
	"time"
)

func ch04send() error {
	var (
		err  error
		n    int
		i    = 7 // 최대 재시도 수
		conn net.Conn
	)
	// 네트워크 연결로의 쓰기 시도는 종종 일시적인 에러가 발생하므로, 재시도가 필요
	// 이를 위한 방법 중 하나 : 쓰기에 관련된 코드를 for 문으로 감싸는 것. 이렇게 하면 필요 시에 쓰기 시도를 재시도하기 쉬워짐
	for ; i > 0; i-- {
		// 네트워크 연결로 쓰기 시도를 위해 다른 io.Writer에 쓰는 것처럼 Write 메서드에 바이트 슬라이스를 매개변수로 전달함
		// Write 메서드는 쓰인 바이트의 숫자와 error 인터페이스를 반환
		n, err = conn.Write([]byte("hello world"))
		if err != nil {
			// error 인터페이스가 nil이 아닌 경우, 타입 어설션을 통해 에러가 net.Error 인터페이스를 구현했는지, 그리고 에러가 일시적인지를 확인
			if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
				// net.Error의 Timeout 메서드가 true를 반환하는 경우
				log.Println("timeout error:", nErr)
				time.Sleep(10 * time.Second)
				// for문을 순회하여 또 다른 쓰기를 시도
				continue
			}
			// 에러가 영구적인 경우, 에러를 반환
			return err
		}
		// 성공적으로 쓰기를 마친 경우, 루프 순회를 종료
		break
	}

	if i == 0 {
		return errors.New("timeout write failure threshold exceeded")
	}

	log.Printf("wrote %d bytes to %s\n", n, conn.RemoteAddr())
	return nil
}
