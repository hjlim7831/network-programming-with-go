package ch03

import (
	"context"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestDialContext(t *testing.T) {
	// 현재 시간으로부터 5초 뒤의 시간 저장
	dl := time.Now().Add(5 * time.Second)
	// 컨텍스트와 cancel 함수 생성, 위에서 생성한 데드라인을 설정
	ctx, cancel := context.WithDeadline(context.Background(), dl)
	// 컨텍스트가 바로 가비지 컬렉션이 되도록 defer로 cancel 함수 호출
	defer cancel()

	var d net.Dialer // DialContext : Dialer의 메서드
	// Dialer의 Control 함수를 오버라이딩
	d.Control = func(_, _ string, _ syscall.RawConn) error {
		// 컨텍스트의 데드라인이 지나도록 충분히 긴 시간 동안 대기
		time.Sleep(5*time.Second + time.Millisecond)
		return nil
	}
	conn, err := d.DialContext(ctx, "tcp", "10.0.0.0:80")
	// 데드라인이 컨텍스트를 제대로 취소했는지 확인
	// 즉, cancel 함수가 제대로 실행되었는지 확인
	if err == nil {
		conn.Close()
		t.Fatal("connection did not time out")
	}
	nErr, ok := err.(net.Error)
	if !ok {
		t.Error(err)
	} else {
		if !nErr.Timeout() {
			t.Errorf("error is not a timeout: %v", err)
		}
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded; actual: %v", ctx.Err())
	}
}
