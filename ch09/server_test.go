package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/awoodbeck/gnp/ch09/handlers"
)

// 요청 -> http.TimeoutHandler(미들웨어) -> handlers.DefaultHandler 함수에서 반환되는 핸들러로 전달
// 멀티플렉서를 사용하는 대신, 모든 요청을 처리하는 하나의 핸들러만을 지정
func TestSimpleHTTPServer(t *testing.T) {
	// Handler, Address 필드는 필수적으로 값을 할당해야 함
	srv := &http.Server{
		Addr: "127.0.0.1:8081",
		Handler: http.TimeoutHandler(
			handlers.DefaultHandler(), 2*time.Minute, ""),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}
	// 서버의 주소에 바인딩된 net.Listener를 생성
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}
	// 리스너의 Serve 메서드를 사용해 요청을 처리
	// Serve 메서드는 서버가 정상적으로 종료된 경우, http.ErrServerClosed을 반환
	go func() {
		err := srv.Serve(l)
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()

	testCases := []struct {
		method   string
		body     io.Reader
		code     int
		response string
	}{
		{http.MethodGet, nil, http.StatusOK, "Hello, friend!"},
		// 핸들러에서 클라이언트의 입력 값을 반드시 이스케이핑 해줄 것
		{http.MethodPost, bytes.NewBufferString("<world>"), http.StatusOK,
			"Hello, &lt;world&gt;!"},
		// handlers.DefaultHandler 함수에서 반환된 핸들러는 HEAD 메서드를 취급하지 않음 -> 405 (Method Not Allowed) 반환
		{http.MethodHead, nil, http.StatusMethodNotAllowed, ""},
	}

	client := new(http.Client)
	path := fmt.Sprintf("http://%s/", srv.Addr)
	for i, c := range testCases {
		// 새로운 요청을 생성하고, 테스트 케이스에서 매개변수 전달받기
		r, err := http.NewRequest(c.method, path, c.body)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		// 요청을 client.Do 메서드로 전달. 서버로부터 응답 받기(resp)
		resp, err := client.Do(r)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}
		// 상태코드 확인
		if resp.StatusCode != c.code {
			t.Errorf("%d: unexpected status code: %q", i, resp.Status)
		}
		// 응답 body 전체 읽기
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}
		// client가 에러를 반환하지 않는다면, 응답 body가 비어있든 무시하든 습관적으로 항상 응답 body 닫아주기!
		// 그렇지 않으면, 무언가 잘못되었을 경우 client가 하위에 존재하는 TCP 연결을 재사용하지 못할 수 있음
		_ = resp.Body.Close()

		if c.response != string(b) {
			t.Errorf("%d: expected %q; actual %q", i, c.response, b)
		}
	}
	// 모든 테스트가 완료된 후, 서버의 Close 메서드 호출
	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSimpleHTTPServerMethods(t *testing.T) {
	srv := &http.Server{
		Addr: "127.0.0.1:8081",
		Handler: http.TimeoutHandler(
			handlers.DefaultMethodsHandler(), 2*time.Minute, ""),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := srv.Serve(l)
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()

	testCases := []struct {
		method   string
		body     io.Reader
		code     int
		response string
	}{
		{http.MethodGet, nil, http.StatusOK, "Hello, friend!"},
		{http.MethodPost, bytes.NewBufferString("<world>"), http.StatusOK,
			"Hello, &lt;world&gt;!"},
		{http.MethodHead, nil, http.StatusMethodNotAllowed, ""},
	}

	client := new(http.Client)
	path := fmt.Sprintf("http://%s/", srv.Addr)
	for i, c := range testCases {
		r, err := http.NewRequest(c.method, path, c.body)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		resp, err := client.Do(r)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		if resp.StatusCode != c.code {
			t.Errorf("%d: unexpected status code: %q", i, resp.Status)
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}
		_ = resp.Body.Close()

		if c.response != string(b) {
			t.Errorf("%d: expected %q; actual %q", i, c.response, b)
		}
	}

	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
}
