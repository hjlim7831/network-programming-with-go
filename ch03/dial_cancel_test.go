package ch03

import (
	"context"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestDialContextCancel(t *testing.T) {
	// 컨텍스트, 컨텍스트를 취소하는 함수 받기
	ctx, cancel := context.WithCancel(context.Background())
	sync := make(chan struct{})

	// 수동으로 컨텍스트를 취소하므로,
	// 클로저를 만들어 별도로 연결 시도를 처리하기 위한 고루틴 시작
	go func() {
		// 고루틴이 끝나면, sync 채널로 신호 전달
		defer func() { sync <- struct{}{} }()

		var d net.Dialer
		d.Control = func(_, _ string, _ syscall.RawConn) error {
			time.Sleep(time.Second)
			return nil
		}
		// 다이얼러가 연결 시도를 하고, 원격 노드와 핸드셰이크가 끝나면
		t.Log("연결 시도")
		conn, err := d.DialContext(ctx, "tcp", "10.0.0.1:80")
		if err != nil {
			t.Log(err)
			return
		}

		conn.Close()
		t.Error("connection did not time out")
	}()
	t.Log("cancel")
	// 컨텍스트를 취소하기 위해 cancel 함수 호출
	cancel()
	<-sync
	// DialContext 메서드는 즉시 nil이 아닌 에러를 반환하고 고루틴을 종료
	// ctx.Err()는 context.Canceled를 반환해야 함
	if ctx.Err() != context.Canceled {
		t.Errorf("expected canceled context; actual: %q", ctx.Err())
	}
}
