package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// http.Get 함수를 이용해 https://time.gov/의 기본 리소스를 조회
// Go의 HTTP 클라이언트는 자동으로 URL의 스키마에 지정된 HTTPS 프로토콜을 변경함
func TestHeadTime(t *testing.T) {
	resp, err := http.Head("https://www.time.gov/")
	if err != nil {
		t.Fatal(err)
	}
	// 비록 응답 body의 내용을 읽진 않지만, 반드시 닫아야 함
	_ = resp.Body.Close()

	now := time.Now().Round(time.Second)
	// 응답을 받은 후, 서버가 응답을 생성한 시간에 대한 정보인 Date 헤더를 받아 옴
	date := resp.Header.Get("Date")
	if date == "" {
		t.Fatal("no Date header received from time.gov")
	}

	dt, err := time.Parse(time.RFC1123, date)
	if err != nil {
		t.Fatal(err)
	}
	// 이 헤더 정보를 이용해, 현재 컴퓨터의 시간과 얼마나 차이나는지를 확인 가능
	// 서버가 헤더를 생성하고, 코드가 헤더를 읽고 처리하는 데 수 나노초 정도의 레이턴시가 발생할 수도 있음
	t.Logf("time.gov: %s (skew %s)", dt, now.Sub(dt))
}

func TestHeadTimeWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead,
		"https://www.time.gov/", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	cancel() // No further need for the context

	now := time.Now().Round(time.Second)
	date := resp.Header.Get("Date")
	if date == "" {
		t.Fatal("no Date header received from time.gov")
	}

	dt, err := time.Parse(time.RFC1123, date)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("time.gov: %s (skew %s)", dt, now.Sub(dt))
}
