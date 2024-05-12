package echo

import (
	"context"
	"fmt"
	"net"
)

// 송신자가 보낸 UDP 패킷을 받아 그대로 에코잉해주는 UDP 서버
// @param1 UDP 서버를 종료할 수 있도록 콘텍스트를 첫 번째 매개변수로 받음
// @param2 호스트:포트 형식으로 구성된 문자열 주소를 두 번째 매개변수로 받음
// return net.Addr, error 인터페이스
// 이후 net.Addr 인터페이스를 이용해 에코 서버에 메시지를 전송
// 에코 서버를 초기화하는 데 실패한 경우, error 인터페이스는 nil이 아닌 에러 값을 반환
func echoServerUDP(ctx context.Context, addr string) (net.Addr, error) {
	// UDP 연결 맺기
	// net.Listen 함수와 유사함. 단 net.PacketConn 인터페이스를 반환함
	s, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("binding to udp %s: %w", addr, err)
	}
	// 고루틴에서 비동기적으로 메시지 에코잉을 관리
	go func() {
		// ctx.Done 채널에서 블로킹 되어있음
		// ctx가 취소되면, Done 채널의 블로킹이 해제되며 서버가 닫히게 되고 상위에 있는 고루틴도 종료됨
		go func() {
			<-ctx.Done()
			_ = s.Close()
		}()

		buf := make([]byte, 1024)
		for {
			// UDP 연결로부터 데이터를 읽기 위해 ReadFrom 메서드에 바이트 슬라이스를 매개변수로 전달함
			// ReadFrom 메서드는 읽은 바이트 수(n), 송신자의 주소(clientAddr), error 인터페이스(err)을 반환함
			// UDP 연결에는 Accept 메서드가 없음 주의! (UDP는 핸드셰이크 과정이 없기 때문.)
			// 메서드로부터 반환된 clientAddr에 의존해 어떤 노드로부터 메시지가 왔는지 확인해야 함
			n, clientAddr, err := s.ReadFrom(buf) // client to server
			if err != nil {
				return
			}
			// UDP 패킷을 전송하기 위해 바이트 슬라이스와 목적지 주소를 연결의 WriteTo 메서드의 매개변수로 전달
			// WriteTo 메서드는 연결로 쓴 바이트 수(_)와 error 인터페이스(err)를 반환
			// 원격 노드와 수립된 세션이 없으므로, 매개변수로 주소 정보를 전달해줘야 함
			// 이미 존재하는 UDP 연결 객체(s)를 활용하면 더 쉽게 메시지 포워딩이 가능
			// 매번 다른 노드로 새로운 연결 객체를 만들 필요 없이, 이미 존재하는 UDP 연결 객체를 이용해 메시지 포워딩 가능
			_, err = s.WriteTo(buf[:n], clientAddr) // server to client
			if err != nil {
				return
			}
		}
	}()

	return s.LocalAddr(), nil
}
