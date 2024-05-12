package echo

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func TestDialUDP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// 클라이언트 측에서의 연결은 UDP를 사용하고도 net.Conn 인터페이스를 이용해 스트림 지향적인 기능을 사용할 수 있음
	// 하지만 UDP 리스너로는 반드시 net.PacketConn 함수를 사용해야 함
	// 클라이언트로 응답을 전송할 목적의 에코 서버 인스턴스를 생성
	serverAddr, err := echoServerUDP(ctx, "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	// net.Dial 함수의 첫 번째 매개변수로 udp를 전달해, 에코 서버에 UDP로 연결을 시도

	client, err := net.Dial("udp", serverAddr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	// UDP에서는 핸드셰이크가 필요하지 않으므로, 이후 아무 트래픽도 받지 않음

	// Send a message to the client from a rogue connection.
	interloper, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	interrupt := []byte("pardon me")
	// 인터로퍼의 연결로부터 클라이언트에게 메시지를 보냄
	n, err := interloper.WriteTo(interrupt, client.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	_ = interloper.Close()

	if len(interrupt) != n {
		t.Fatalf("wrote %d bytes of %d", n, len(interrupt))
	}

	// Now write a message to the server that will prompt a reply.
	ping := []byte("ping")
	// 클라이언트는 net.Conn 인터페이스의 Write 메서드를 이용해 에코 서버로 ping 메시지를 보냄
	// net.Conn 클라이언트는 net.Dial 함수에서 연결한 주소로 메시지를 전송
	// 패킷 전송 시마다 이 클라이언트 연결을 이용하므로, 목적지 주소를 지정해주지 않아도 됨
	_, err = client.Write(ping)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	// 이 클라이언트의 Read 메서드를 사용해 패킷을 읽음
	n, err = client.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	// The first message the client reads should be the "ping" from the echo
	// server, not the queued up "test" message.
	// 클라이언트는 마치 스트림 지향적인 연결 객체에서 읽는 것처럼 net.Dial 함수에서 지정된 송신자 주소에서 온 패킷만 읽음
	// 인터로퍼가 보낸 메시지는 절대로 읽지 않음
	if !bytes.Equal(ping, buf[:n]) {
		t.Errorf("expected reply %q; actual reply %q", ping, buf[:n])
	}

	// Verify no other incoming packets are waiting.
	// 다른 곳에서 보낸 메시지를 읽지는 않는지, 데드라인을 길게 설정하고
	err = client.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	// 다른 메시지를 한 번 읽어보기
	_, err = client.Read(buf)
	if err == nil {
		t.Fatal("unexpected packet")
	}
}
