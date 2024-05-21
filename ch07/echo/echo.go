package echo

import (
	"context"
	"fmt"
	"net"
	"os"
)

// 스트림 기반의 네트워크를 나타내는 문자열과 주소를 나타내는 문자열을 매개변수로 받음
// 생성된 주소 객체와 error 인터페이스를 반환
// 콘텍스트와 네트워크 문자열, 주소 문자열을 받는 조금 더 일반적인 형태의 에코 서버를 만듦
// -> tcp, unix, unixpacket과 같은 스트림 기반의 네트워크 타입을 네트워크 문자열로 전달 가능
// 네트워크 타입에 따라 주소 문자열도 적용하면 됨
// tcp -> IP주소:포트 / unix, unixpacket -> 존재하지 않는 파일 경로
// 콘텍스트 : 서버 종료를 알리는 시그널링을 위해 사용됨
func streamingEchoServer(ctx context.Context, network string,
	addr string) (net.Addr, error) {
	// 에코 서버가 바인딩하게 되면, 소켓 파일이 생성됨
	s, err := net.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("binding to %s %s: %w", network, addr, err)
	}

	go func() {
		go func() {
			// 함수 호출자가 콘텍스트를 취소하면, 서버는 종료됨
			<-ctx.Done()
			_ = s.Close()
		}()
		// 연결 요청 수신 대기
		for {
			// 서버가 연결을 수신하면
			conn, err := s.Accept()
			if err != nil {
				return
			}
			// 수신받는 메시지를 별도의 고루틴에서 에코잉함
			go func() {
				defer func() { _ = conn.Close() }()

				for {
					buf := make([]byte, 1024)
					n, err := conn.Read(buf)
					if err != nil {
						return
					}

					_, err = conn.Write(buf[:n])
					if err != nil {
						return
					}
				}
			}()
		}
	}()

	return s.Addr(), nil
}

func datagramEchoServer(ctx context.Context, network string,
	addr string) (net.Addr, error) {
	// net.PacketConn 객체를 반환하는 net.ListenPacket 함수를 호출
	s, err := net.ListenPacket(network, addr)
	if err != nil {
		return nil, fmt.Errorf("binding to %s %s: %w", network, addr, err)
	}

	go func() {
		go func() {
			<-ctx.Done()
			_ = s.Close()
			// 코드 상에서 반드시 소켓 파일을 직접 지워야 함
			// 그렇지 않으면, 동일한 소켓 파일 경로로 시도하는 바인딩이 모두 실패할 것
			if network == "unixgram" {
				_ = os.Remove(addr)
			}
		}()

		buf := make([]byte, 1024)
		for {
			n, clientAddr, err := s.ReadFrom(buf)
			if err != nil {
				return
			}

			_, err = s.WriteTo(buf[:n], clientAddr)
			if err != nil {
				return
			}
		}
	}()

	return s.LocalAddr(), nil
}
