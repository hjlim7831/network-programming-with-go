package ch03

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestDialContextCancelFanOut(t *testing.T) {
	// context.WithDeadline 함수로 생성된 ctx는
	// Err 메서드로 context.Canceled, context.DeadlineExceeded, nil
	// 3개 중 하나의 값을 반환함
	ctx, cancel := context.WithDeadline(
		context.Background(),
		time.Now().Add(10*time.Second),
	)
	// 리스너 생성
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	go func() {
		// 리스너는 하나의 연결을 수락
		// 이 때, 이후로 연결 요청이 온 것들과 3 way handshake까지는 마침
		conn, err := listener.Accept()
		// 성공적으로 연결을 수락한 뒤, 연결을 종료
		if err == nil {
			conn.Close()
		}
	}()
	// 다이얼러 생성
	// 여러 개의 다이얼러를 실행하므로
	// 다이얼링을 위한 코드를 추상화해 별도의 함수로 분리
	dial := func(ctx context.Context, address string, response chan int,
		id int, wg *sync.WaitGroup) {
		// WaitGroup을 이용해 콘텍스트를 취소하여
		// for 루프에서 생성한 모든 다이얼 고루틴을 정상적으로 종료
		defer wg.Done()

		var d net.Dialer
		// DialContext 함수를 이용해 매개변수로 주어진 주소로 연결을 시도
		c, err := d.DialContext(ctx, "tcp", address)
		if err != nil {
			return
		}
		c.Close()
		// 연결이 성공하면 아직 콘텍스트를 취소하지 않았다고 생각하고
		// 다이얼러의 ID를 응답 채널에 전송
		select {
		case <-ctx.Done():
		case response <- id: // 여기서 response는 버퍼가 없는 채널. id가 1개가 넘어오면 끝
		}
	}

	res := make(chan int)
	var wg sync.WaitGroup
	// 별도의 고루틴을 호출해 여러 개의 다이얼러를 생성
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go dial(ctx, listener.Addr().String(), res, i+1, &wg)
	}
	// 고루틴이 정상적으로 동작하면, 한 연결 시도가 먼저 성공적으로 리스너에 연결될 수 있음

	// 연결이 성공한 다이얼러의 ID를 res 채널에서 받음
	response := <-res

	// cancel 함수로 다이얼러에 대한 연결을 중단함
	// Err 메서드는 context.Canceled을 반환할 것
	cancel()

	// 다른 다이얼러들의 연결 시도 중단이 끝나고, 고루틴이 종료될 때까지 블로킹됨
	wg.Wait()
	close(res)

	// 발생한 컨텍스트 취소가 코드상의 취소였음을 확인
	if ctx.Err() != context.Canceled {
		t.Errorf("expected canceled context; actual: %s",
			ctx.Err(),
		)
	}

	t.Logf("dialer %d retrieved the resource", response)
}
