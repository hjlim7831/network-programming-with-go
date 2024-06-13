package ch11

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEchoServerTLS(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverAddress := "localhost:34443"
	maxIdle := time.Second
	server := NewTLSServer(ctx, serverAddress, maxIdle, nil)
	done := make(chan struct{})

	go func() {
		// 백그라운드에서 서버 시작
		err := server.ListenAndServeTLS("cert.pem", "key.pem")
		if err != nil && !strings.Contains(err.Error(),
			"use of closed network connection") {
			t.Error(err)
			return
		}
		done <- struct{}{}
	}()
	// 수신 연결 준비가 완료될 때까지 블로킹
	server.Ready()

	// cert.pem 파일 읽기
	cert, err := os.ReadFile("cert.pem")
	if err != nil {
		t.Fatal(err)
	}

	// 새로운 인증서 풀을 생성
	certPool := x509.NewCertPool()
	// 인증서 풀에 인증서를 추가
	if ok := certPool.AppendCertsFromPEM(cert); !ok {
		t.Fatal("failed to append certificate to pool")
	}

	// 인증서 풀을 tls.Config의 RootCAs 필드에 추가
	// 이를 이용하면 아직 기존 인증서의 만료 기간이 일부 남은 상화에서 새로운 인증서로 마이그레이션하는 데 유용하게 사용 가능
	// 이 config로 생성된 클라이언트는 TLS negotitation에서 cert.pem 인증서를 사용한 서버만, 혹은 cert.pem 인증서로 서명된 인증서를 사용한 서버만을 인증함
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		RootCAs:          certPool,
	}

	// tls.Dial에 tls.Config 객체를 매개변수로 전달
	// 이로 인해 TLS 클라이언트는 서버의 인증서가 사설 인증서로 서명되었음에도, InsecureSkipVerify 필드를 설정하거나 보안에 취약한 어떠한 옵션도 사용하지 않고 서버의 인증서를 인증할 수 있음
	conn, err := tls.Dial("tcp", serverAddress, tlsConfig)
	if err != nil {
		t.Fatal(err)
	}

	hello := []byte("hello")
	_, err = conn.Write(hello)
	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	if actual := b[:n]; !bytes.Equal(hello, actual) {
		t.Fatalf("expected %q; actual %q", hello, actual)
	}

	// 서버가 긴 시간 유휴 상태로 있으면
	time.Sleep(2 * maxIdle)
	_, err = conn.Read(b)
	// 소켓과의 통신에서 서버가 소켓을 닫았다는 에러가 발생할 것
	if err != io.EOF {
		t.Fatal(err)
	}

	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	<-done
}
