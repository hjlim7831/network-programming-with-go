package main

import (
	"crypto/rand"
	"io"
	"net"
	"testing"
)

func TestReadIntoBuffer(t *testing.T) {
	payload := make([]byte, 1<<24) // 16 MB
	// 클라이언트가 읽어 들일 16 MB 페이로드의 랜덤 데이터를 생성
	_, err := rand.Read(payload)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		// 리스너가 연결되면
		conn, err := listener.Accept()
		if err != nil {
			t.Log(err)
			return
		}
		defer conn.Close()

		// 서버는 네트워크 연결(conn)로 페이로드 전체를 씀
		_, err = conn.Write(payload)
		if err != nil {
			t.Error(err)
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// 클라이언트의 512 KB 버퍼에서 읽어 들일 수 있는 데이터의 양보다 더 많음
	buf := make([]byte, 1<<19) // 512 KB
	// for 문에서 몇 번 반복적으로 순회해서 읽어 들여야 함
	// 더 큰 버퍼를 사용하거나, 더 작은 페이로드를 사용해 페이로드 전체를 한 번의 Read 메서드 호출로 전부 다 읽어 들이는 것도 좋음
	for {
		n, err := conn.Read(buf)
		// 연결에서 에러가 반환되거나, 16MB 페이로드 전체를 읽어 들일 때까지(io.EOF) 반복해서 데이터 읽음
		if err != nil {
			if err != io.EOF {
				t.Error(err)
			}
			break
		}
		// 읽을 때 전부 다 512KB 만큼 읽어지지는 않음
		t.Logf("read %d bytes", n) // buf[:n] is the data read from conn
	}

	conn.Close()
}
