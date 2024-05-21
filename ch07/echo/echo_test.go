package echo

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestEchoServerUnix(t *testing.T) {
	// 운영체제의 임시 디렉터리에 echo_unix라는 하위 디렉터리 생성
	dir, err := os.MkdirTemp("", "echo_unix")
	if err != nil {
		t.Fatal(err)
	}
	// 테스트가 종료되면, 임시 디렉터리를 삭제
	defer func() {
		if rErr := os.RemoveAll(dir); rErr != nil {
			t.Error(rErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	socket := filepath.Join(dir, fmt.Sprintf("%d.sock", os.Getpid()))
	// 소켓 파일 이름을 #.socket을 전달 (# : processId)
	rAddr, err := streamingEchoServer(ctx, "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	// 모든 사용자가 소켓에 읽기, 쓰기 권한이 있는지 확인
	err = os.Chmod(socket, os.ModeSocket|0666)
	if err != nil {
		t.Fatal(err)
	}

	// 서버에 다이얼링
	// 네트워크 타입 : unix
	// 유닉스 도메인 소켓 파일의 전체 경로를 서버의 주소로 전달
	t.Logf("unix domain socket address: %s", rAddr)
	conn, err := net.Dial("unix", rAddr.String())
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("ping")
	// 첫 번째 응답을 읽기 전, 에코 서버로 3 개의 ping 메시지를 전송
	for i := 0; i < 3; i++ {
		_, err = conn.Write(msg)
		if err != nil {
			t.Fatal(err)
		}
	}
	// 전송한 3 개의 메시지를 읽기에 충분한 버퍼에 첫 번째 응답을 읽으면
	buf := make([]byte, 1024)
	n, err := conn.Read(buf) // read once from the server
	if err != nil {
		t.Fatal(err)
	}
	// 스트림 기반 연결에는 메시지의 구분자가 없음
	// 서버의 바이트 스트림으로부터 하나의 메시지가 시작하고 끝나는 지점을 읽고 구분하는 것은 코드상에서 해야 하는 일
	t.Logf("actual: %s", string(buf[:n]))

	expected := bytes.Repeat(msg, 3)
	if !bytes.Equal(expected, buf[:n]) {
		t.Fatalf("expected reply %q; actual reply %q", expected,
			buf[:n])
	}
}

func BenchmarkEchoServerUDP(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr, err := datagramEchoServer(ctx, "udp", "127.0.0.1:")
	if err != nil {
		b.Fatal(err)
	}
	defer cancel()

	client, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	msg := []byte("ping")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = client.WriteTo(msg, serverAddr)
		if err != nil {
			b.Fatal(err)
		}

		buf := make([]byte, 1024)
		n, addr, err := client.ReadFrom(buf)
		if err != nil {
			b.Fatal(err)
		}

		if addr.String() != serverAddr.String() {
			b.Fatalf("received reply from %q instead of %q", addr,
				serverAddr)
		}

		if !bytes.Equal(msg, buf[:n]) {
			b.Fatalf("expected reply %q; actual reply %q", msg, buf[:n])
		}
	}
}

func BenchmarkEchoServerTCP(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	rAddr, err := streamingEchoServer(ctx, "tcp", "127.0.0.1:")
	if err != nil {
		b.Fatal(err)
	}
	defer cancel()

	conn, err := net.Dial("tcp", rAddr.String())
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	msg := []byte("ping")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = conn.Write(msg)
		if err != nil {
			b.Fatal(err)
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			b.Fatal(err)
		}

		if !bytes.Equal(msg, buf[:n]) {
			b.Fatalf("expected reply %q; actual reply %q", msg, buf[:n])
		}
	}
}

func BenchmarkEchoServerUnix(b *testing.B) {
	dir, err := os.MkdirTemp("", "echo_unix_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if rErr := os.RemoveAll(dir); rErr != nil {
			b.Error(rErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	socket := filepath.Join(dir, fmt.Sprintf("%d.sock", os.Getpid()))
	rAddr, err := streamingEchoServer(ctx, "unix", socket)
	if err != nil {
		b.Fatal(err)
	}
	defer cancel()

	conn, err := net.Dial("unix", rAddr.String())
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	msg := []byte("ping")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = conn.Write(msg)
		if err != nil {
			b.Fatal(err)
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			b.Fatal(err)
		}

		if !bytes.Equal(msg, buf[:n]) {
			b.Fatalf("expected reply %q; actual reply %q", msg, buf[:n])
		}
	}
}

func TestEchoServerUDP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr, err := datagramEchoServer(ctx, "udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	client, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	msg := []byte("ping")
	_, err = client.WriteTo(msg, serverAddr)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, addr, err := client.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}

	if addr.String() != serverAddr.String() {
		t.Fatalf("received reply from %q instead of %q", addr, serverAddr)
	}

	if !bytes.Equal(msg, buf[:n]) {
		t.Fatalf("expected reply %q; actual reply %q", msg, buf[:n])
	}
}

func TestEchoServerTCP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	rAddr, err := streamingEchoServer(ctx, "tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	conn, err := net.Dial("tcp", rAddr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	msg := []byte("ping")
	_, err = conn.Write(msg)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(msg, buf[:n]) {
		t.Fatalf("expected reply %q; actual reply %q", msg, buf[:n])
	}
}
