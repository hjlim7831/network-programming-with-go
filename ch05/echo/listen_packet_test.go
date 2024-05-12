package echo

import (
	"bytes"
	"context"
	"net"
	"testing"
)

func TestListenPacketUDP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// 에코 서버 생성
	serverAddr, err := echoServerUDP(ctx, "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	// 클라이언트 생성
	client, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// 클라이언트와 에코 서버 간에 끼어들 새로운 UDP 연결(interloper)을 생성
	interloper, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	interrupt := []byte("pardon me")
	//클라이언트를 인터럽트함. 이 메시지는 클라이언트의 수신 버퍼에 큐잉됨
	n, err := interloper.WriteTo(interrupt, client.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	_ = interloper.Close()

	if l := len(interrupt); l != n {
		t.Fatalf("wrote %d bytes of %d", n, l)
	}

	ping := []byte("ping")
	// 클라이언트는 에코 서버로 ping 메시지를 쓰고
	_, err = client.WriteTo(ping, serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	// 수신하는 메시지를 즉시 읽음
	n, addr, err := client.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}

	// UDP 연결에서 독특한 점
	// 서버와 클라이언트 사이에 끼어든 연결로부터 인터럽트 메시지를 먼저 읽은 후
	// (TCP 연결이었다면 인터로퍼가 존재하지 않았을 것)
	if !bytes.Equal(interrupt, buf[:n]) {
		t.Errorf("expected reply %q; actual reply %q", interrupt, buf[:n])
	}

	if addr.String() != interloper.LocalAddr().String() {
		t.Errorf("expected message from %q; actual sender is %q",
			interloper.LocalAddr(), addr)
	}

	n, addr, err = client.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}

	// 에코 서버로 응답함
	if !bytes.Equal(ping, buf[:n]) {
		t.Errorf("expected reply %q; actual reply %q", ping, buf[:n])
	}
	// 그래서 애플리케이션상에서 읽어 들이는 패킷마다 ReadFrom 메서드의 두 번째 반환값을 평가하여 송신자의 주소를 검증해야 함
	if addr.String() != serverAddr.String() {
		t.Errorf("expected message from %q; actual sender is %q",
			serverAddr, addr)
	}
}
