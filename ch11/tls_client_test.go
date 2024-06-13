package ch11

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

func TestClientTLS(t *testing.T) {
	// HTTPS 서버를 반환
	// 새로운 인증서 생성을 포함해, HTTPS 서버 초기화를 위한 TLS 세부 환경구성까지 처리해줌
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// 서버에서 HTTP로 클라이언트 요청을 받으면, 요청 객체의 TLS 필드는 nil이 됨

				if r.TLS == nil {
					// 이러한 케이스를 확인하고, 클라이언트의 요청을 HTTPS로 리다이렉트 시킴
					u := "https://" + r.Host + r.RequestURI
					http.Redirect(w, r, u, http.StatusMovedPermanently)
					return
				}

				w.WriteHeader(http.StatusOK)
			},
		),
	)
	defer ts.Close()

	// 테스트를 위해, 서버 객체의 Client 메서드는 서버의 인증서를 신뢰하는 *http.Client 객체를 반환
	// 이 클라이언트를 이용해, 핸들러 내의 TLS와 관련된 코드를 테스트할 수 있음
	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d; actual status %d",
			http.StatusOK, resp.StatusCode)
	}

	// 새로운 트랜스포트를 생성하고 TLS 구성을 정의하기
	// Transport의 TLS 구성을 오버라이딩할 경우, HTTP/2 지원이 제거됨
	tp := &http.Transport{
		TLSClientConfig: &tls.Config{
			// P-256이 P-384나 P-521보다 좋음
			// (1) P-256은 나머지 두 개와 달리, 시간차 공격에 저항이 있음
			// (2) P-256을 사용하면 클라이언트는 TLS negotitation에서 최소 1.2 이상의 버전을 사용함
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			MinVersion:       tls.VersionTLS12,
		},
	}

	// 이 트랜스포트를 사용하도록 http2를 구성
	err = http2.ConfigureTransport(tp)
	if err != nil {
		t.Fatal(err)
	}

	// 클라이언트 트랜스포트의 기본 TLS 구성을 오버라이딩
	client2 := &http.Client{Transport: tp}

	// 명시적으로 신뢰할 인증서를 선택하지 않으면, 클라이언트는 운영체제가 신뢰하는 인증 저장소의 인증서를 신뢰함
	// 테스트 서버로 보내는 첫 번째 요청은 클라이언트가 서버가 보내는 인증서의 서명자를 신뢰하지 않으므로, 실패해 에러가 발생
	_, err = client2.Get(ts.URL)
	if err == nil || !strings.Contains(err.Error(),
		"certificate signed by unknown authority") {
		t.Fatalf("expected unknown authority error; actual: %q", err)
	}

	// 위의 상황을 우회하기 위해, InsecureSkipVerify 필드의 값을 true로 설정
	// 클라이언트가 서버의 인증서를 검증하지 않도록 할 수 있음
	// 디버깅 외의 목적으로 이 값을 사용하는 것은 보안상의 이유로 추천하지 않음
	tp.TLSClientConfig.InsecureSkipVerify = true

	resp, err = client2.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d; actual status %d",
			http.StatusOK, resp.StatusCode)
	}
}

func TestClientTLSGoogle(t *testing.T) {
	conn, err := tls.DialWithDialer(
		// *net.Dialer 객체의 포인터
		&net.Dialer{Timeout: 30 * time.Second},
		// 네트워크 종류
		"tcp",
		// 네트워크 주소
		"www.google.com:443",
		&tls.Config{
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			MinVersion:       tls.VersionTLS12,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// dial에 성공하면, TLS 연결의 세부 상태 정보를 탐색할 수 있음
	state := conn.ConnectionState()
	t.Logf("TLS 1.%d", state.Version-tls.VersionTLS10)
	t.Log(tls.CipherSuiteName(state.CipherSuite))
	t.Log(state.VerifiedChains[0][0].Issuer.Organization[0])

	_ = conn.Close()
}
