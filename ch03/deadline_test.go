package ch03

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestDeadline(t *testing.T) {
	sync := make(chan struct{})

	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Log(err)
			return
		}
		defer func() {
			conn.Close()
			close(sync) // 이른 return으로 sync 채널에서 읽는 데이터가 블로킹되면 안 됨
		}()
		// 데드라인 설정
		// 클라이언트가 데이터를 전송하지 않으므로, Read 메서드는 데드라인이 지날 때까지 블로킹될 것
		err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			t.Error(err)
			return
		}

		buf := make([]byte, 1)
		_, err = conn.Read(buf) // 원격 노드가 데이터를 보낼 때까지 블로킹되지만, 데이터를 보내는 로직 없음
		nErr, ok := err.(net.Error)
		// 에러 반환. 이후에 발생하는 모든 읽기 시도는 다른 타임아웃 에러를 반환할 것
		if !ok || !nErr.Timeout() {
			t.Errorf("expected timeout error; actual: %v", err)
		}
		// 첫 번째 Read에서 에러를 처리하고 이젠 데이터를 보내도록 메인 테스트 루틴에게 신호를 보냄
		sync <- struct{}{}
		// 데드라인을 좀 더 뒤로 설정해 다시 읽기가 정상적으로 동작하게 할 수 있음
		err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			t.Error(err)
			return
		}
		// Read 메서드 성공
		_, err = conn.Read(buf)
		if err != nil {
			t.Error(err)
		}
	}()

	// 클라이언트 측
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// sync로부터 신호를 받을 때까지 블로킹됨
	<-sync
	_, err = conn.Write([]byte("1"))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1)
	_, err = conn.Read(buf) // 원격 노드가 데이터를 보낼 때까지 블로킹됨 (하지만 보내지 않지)
	// 여기서 Read 메서드에서 블로킹된 클라이언트는
	// 네트워크 연결이 종료됨에 따라 io.EOF를 받음
	if err != io.EOF {
		t.Errorf("expected server termination; actual: %v", err)
	}
}
