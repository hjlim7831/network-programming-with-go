package ch03

import (
	"context"
	"io"
	"time"
)

const defaultPingInterval = 30 * time.Second

// 일정한 간격마다 핑 메시지를 전송하는 함수
func Pinger(ctx context.Context, w io.Writer, reset <-chan time.Duration) {
	var interval time.Duration
	select {
	case <-ctx.Done():
		return
	// 타이머 초기 간격 설정을 위해 버퍼 채널을 생성해 대기 시간을 설정
	case interval = <-reset: // reset 채널에서 초기 간격을 받아 옴
	default:
	}
	// 시간 간격이 0 미만이면 기본 핑 시간 간격을 사용
	if interval <= 0 {
		interval = defaultPingInterval
	}
	// timer를 interval로 초기화
	timer := time.NewTimer(interval)
	// 필요한 경우 defer를 사용해 타이머의 채널의 값을 소비
	defer func() {
		if !timer.Stop() {
			<-timer.C // 타이머가 만료될 때까지 대기
		}
	}()
	// 종료되지 않는 for문
	// 아래의 3가지 중 하나가 일어날 때까지 블로킹됨
	for {
		select {
		// 1. 콘텍스트가 취소되거나
		case <-ctx.Done():
			// 이 경우 함수 종료. 더 이상의 핑은 전송되지 않음
			return
		// 2. 타이머를 리셋하기 위한 시그널을 받았거나
		case newInterval := <-reset:
			// timer의 카운트다운을 중지시키는 메서드.
			if !timer.Stop() {
				// 만약 false를 반환한다면
				<-timer.C // 채널에 남아있는 이벤트를 수동으로 제거. (채널의 값을 소비해 채널을 비움)
			}
			if newInterval > 0 {
				interval = newInterval
			}
		// 3. 타이머가 만료되었거나
		case <-timer.C:
			// 핑 메시지를 timer에 쓰고
			if _, err := w.Write([]byte("ping")); err != nil {
				// 여기서 연속으로 발생하는 타임아웃을 추적하고 처리
				// 이를 위해 컨텍스트의 cancel 함수를 전달
				// 연속적 타임아웃이 임계값을 넘게 되면, cancel 함수를 호출
				return
			}
		}
		// 타이머 리셋
		_ = timer.Reset(interval)
	}
}
