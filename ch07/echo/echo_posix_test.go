//go:build darwin || linux
// +build darwin linux

// go가 패키지를 빌드할 때 build constraint를 확인해 이 코드가 macOS와 리눅스 운영체제에서만 동작하도록 함

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

func TestEchoServerUnixDatagram(t *testing.T) {
	dir, err := os.MkdirTemp("", "echo_unixgram")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		// 여기서 임시 디렉터리의 하위 디렉터리를 지울 때
		// 소켓 파일도 함께 제거됨
		if rErr := os.RemoveAll(dir); rErr != nil {
			t.Error(rErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	// 서버의 소켓 파일
	sSocket := filepath.Join(dir, fmt.Sprintf("s%d.sock", os.Getpid()))
	serverAddr, err := datagramEchoServer(ctx, "unixgram", sSocket)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	err = os.Chmod(sSocket, os.ModeSocket|0622)
	if err != nil {
		t.Fatal(err)
	}
	// 클라이언트의 소켓 파일
	cSocket := filepath.Join(dir, fmt.Sprintf("c%d.sock", os.Getpid()))
	client, err := net.ListenPacket("unixgram", cSocket)
	if err != nil {
		t.Fatal(err)
	}
	// 여기서 os.Remove 함수를 호출해 클라이언트의 소켓 파일을 지울 수도 있음
	defer func() { _ = client.Close() }()

	// 또, 서버는 클라이언트의 소켓 파일에도 쓸 수 있어야 함
	// 이 예시에서는 굉장히 허용적인 권한을 줌
	// 모든 사용자가 소켓에 쓰기 권한을 가짐
	err = os.Chmod(cSocket, os.ModeSocket|0622)
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("ping")
	for i := 0; i < 3; i++ { // write 3 "ping" messages
		_, err = client.WriteTo(msg, serverAddr)
		if err != nil {
			t.Fatal(err)
		}
	}

	buf := make([]byte, 1024)

	// unixgram 소켓은 각 메시지를 구분하여 관리
	for i := 0; i < 3; i++ { // read 3 "ping" messages
		n, addr, err := client.ReadFrom(buf)
		if err != nil {
			t.Fatal(err)
		}

		if addr.String() != serverAddr.String() {
			t.Fatalf("received reply from %q instead of %q", addr,
				serverAddr)
		}

		if !bytes.Equal(msg, buf[:n]) {
			t.Fatalf("expected reply %q; actual reply %q", msg, buf[:n])
		}
	}
}

func BenchmarkEchoServerUnixDatagram(b *testing.B) {
	dir, err := os.MkdirTemp("", "echo_unixgram_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if rErr := os.RemoveAll(dir); rErr != nil {
			b.Error(rErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	socket := filepath.Join(dir, fmt.Sprintf("s%d.sock", os.Getpid()))
	serverAddr, err := datagramEchoServer(ctx, "unixgram", socket)
	if err != nil {
		b.Fatal(err)
	}
	defer cancel()

	cSocket := filepath.Join(dir, fmt.Sprintf("c%d.sock", os.Getpid()))
	client, err := net.ListenPacket("unixgram", cSocket)
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
