package echo

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestEchoServerUDP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// echoServer 함수에 매개변수로 콘텍스트와 주소의 문자열을 전달하고 서버의 주소를 반환받음
	serverAddr, err := echoServerUDP(ctx, "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	// 콘텍스트 취소 함수를 defer로 호출시켜, 함수가 종료되면 서버 또한 종료되도록 함
	// 실제 애플리케이션에서는 콘텍스트를 사용하여 장기적으로 실행되는 프로세스를 취소하면 메모리와 같은 자원이 낭비되지 않도록 하며, 불필요하게 파일 디스크립터 핸들을 연 채로 유지하지 않도록 함
	defer cancel()

	// 클라이언트의 net.PacketConn 인터페이스 초기화
	// 클라이언트와 서버 양측의 연결 객체를 생성 (client)
	client, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	msg := []byte("ping")
	// 클라이언트에게 메시지를 어디로 전송할 것인지, WriteTo 메서드를 호출할 때마다 매개변수로 주소를 전달해 줘야 함
	_, err = client.WriteTo(msg, serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	// 에코 서버로 메시지를 전송한 후, 클라이언트는 ReadFrom 메서드를 통해 즉시 메시지를 읽을 수 있음
	n, addr, err := client.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	// ReadFrom 메서드에서 반환된 주소(addr)를 사용해 에코 서버가 메시지를 보냈는지 확인할 수 있음
	if addr.String() != serverAddr.String() {
		t.Fatalf("received reply from %q instead of %q", addr, serverAddr)
	}

	if !bytes.Equal(msg, buf[:n]) {
		t.Errorf("expected reply %q; actual reply %q", msg, buf[:n])
	}
}

func TestDropLocalhostUDPPackets(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer cancel()

		buf := make([]byte, 1024)
		for {
			n, clientAddr, err := s.ReadFrom(buf) // client to server
			if err != nil {
				return
			}

			_, err = s.WriteTo(buf[:n], clientAddr) // server to client
			if err != nil {
				return
			}
		}
	}()

	server, ok := s.(*net.UDPConn)
	if !ok {
		t.Fatal("not a UDPConn")
	}
	err = server.SetWriteBuffer(2)
	if err != nil {
		t.Fatal(err)
	}

	lAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	client, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	err = client.SetReadBuffer(2)
	if err != nil {
		t.Fatal(err)
	}

	pings := 50
	for i := 0; i < pings; i++ {
		msg := []byte(fmt.Sprintf("%2d", i))
		_, err = client.WriteTo(msg, s.LocalAddr())
		if err != nil {
			t.Fatal(err)
		}
	}

	err = client.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		t.Fatal(err)
	}

	recv := make(chan []byte)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, _, err := client.ReadFrom(buf)
			if err != nil {
				_ = s.Close()
			}
			recv <- buf[:n]
		}
	}()

	replies := 0
OUTER:
	for {
		select {
		case m := <-recv:
			replies++
			t.Logf("%s", m)
		case <-ctx.Done():
			break OUTER
		}
	}

	if replies >= pings {
		t.Fatal("no replies were dropped")
	}
	t.Logf("received %d replies", replies)
}
