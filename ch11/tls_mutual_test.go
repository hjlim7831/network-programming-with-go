package ch11

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
)

func caCertPool(caCertFn string) (*x509.CertPool, error) {
	// PEM 포맷으로 인코딩된 인증서 파일의 경로를 매개변수로 받은 뒤, 파일의 내용을 읽고
	caCert, err := os.ReadFile(caCertFn)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	// 읽은 내용을 새로운 인증서 풀에 추가
	if ok := certPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.New("failed to add certificate to pool")
	}
	// 이 인증서 풀은 신뢰하는 인증서의 소스로 사용됨
	// 클라이언트는 서버의 인증서를, 서버는 클라이언트의 인증서를 풀에 추가함
	return certPool, nil
}

func TestMutualTLSAuthentication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 서버 생성 전, 먼저 클라이언트의 인증서를 이용해 새로운 CA 인증서 풀을 생성해야 함
	serverPool, err := caCertPool("clientCert.pem")
	if err != nil {
		t.Fatal(err)
	}

	// ServeTLS 메서드를 사용해 서버의 인증서를 로드
	cert, err := tls.LoadX509KeyPair("serverCert.pem", "serverKey.pem")
	if err != nil {
		t.Fatalf("loading key pair: %v", err)
	}

	// 상호 TLS 인증을 구현함에 있어, 클라이언트 인증서의 CN 값이나 SAN 값으로부터 클라이언트의 IP 주소 혹은 호스트 네임을 인증하기 위해 서버의 인증서 검증 과정에 일부 변화를 줘야 함
	// 이를 위해, 최소한 서버에서는 클라이언트의 IP 주소를 알고 있어야 함
	// tls.Config 객체의 GetConfigForClient 메서드를 정의하는 것만이 인증서 검증 이전에 클라이언트의 연결 정보를 얻을 수 있는 유일한 방법
	// 이 메서드는 클라이언트와 TLS 핸드셰이크 프로세스 과정에서 생성되는 tls.ClientHelloInfo 포인터 객체를 매개변수로 받는 함수를 정의할 수 있게 해 줌
	// 이 메서드를 사용해 클라이언트의 IP 주소를 얻어올 수 있음
	serverConfig := &tls.Config{
		// TLS 구성에 서버의 인증서 정보를 추가
		Certificates: []tls.Certificate{cert},
		// 이 함수는 모든 클라이언트 연결에 동일한 TLS 구성을 반환함
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config,
			error) {
			return &tls.Config{
				Certificates: []tls.Certificate{cert},
				// 서버에서 TLS 핸드셰이크 프로세스를 완료하기 전에, 모든 클라이언트가 유효한 인증서를 제공하였는지 확인하기
				ClientAuth: tls.RequireAndVerifyClientCert,
				// 서버의 인증서 pool 정보 추가
				ClientCAs:        serverPool,
				CurvePreferences: []tls.CurveID{tls.CurveP256},
				// 필요한 최소 TLS 버전을 1.3으로 지정
				MinVersion:               tls.VersionTLS13,
				PreferServerCipherSuites: true,
				// 서버의 일반적인 인증서 검증 절차를 강화하기 위해 필요한 함수를 정의하고, TLS 구성 객체의 VerifyPeerCertificate 메서드 필드에 할당해주기
				// 서버는 일반적인 인증서 검증 확인 이후, 이 메서드를 호출
				// 일반적인 인증서 검사 이후에 수행하는 확인 절차에는 leaf certificate를 이용해 클라이언트의 호스트 네임을 검증하는 작업이 있음
				// leaf certificate : 클라이언트가 서버에 제출한 인증서 체인의 제일 마지막에 존재하는 인증서
				// 여기에는 클라이언트의 공개키가 포함되어 있음
				// 인증서 체인에 존재하는 leaf certificate 외의 인증서에는 최상단 인증 기관(root CA) 인증서에 다다르기까지 리프 인증서의 진위 여부를 검증해주는 중간 인증서들로 구성됨
				// 각각의 verifiedChains 슬라이서의 0번 인덱스에 leaf certificate가 존재함
				// ex) 첫 번째 체인의 leaf certificate : verifiedChains[0][0]
				// 만약 서버에서 VerifyPeerCertificate 메서드에 할당된 함수를 호출하면, 최소한 체인의 첫번째에 leaf certificate가 존재하게 됨
				VerifyPeerCertificate: func(rawCerts [][]byte,
					verifiedChains [][]*x509.Certificate) error {

					opts := x509.VerifyOptions{
						// x509.VerifyOptions 객체를 생성하고, 클라이언트 인증을 위한 KeyUsages 메서드를 수정
						KeyUsages: []x509.ExtKeyUsage{
							x509.ExtKeyUsageClientAuth,
						},
						// Roots 메서드에 서버 풀을 할당
						// 서버는 검증 단계에서 pool에 등록된 인증서를 신뢰함
						Roots: serverPool,
					}
					// hello에서 클라이언트의 연결 객체를 이용해 클라이언트의 IP 주소를 얻어옴
					ip := strings.Split(hello.Conn.RemoteAddr().String(),
						":")[0]
					// 얻어온 IP를 사용해 역 DNS 추적을 하여 해당하는 IP에 할당된 호스트 네임이 있는지 조회
					// 이 추적이 실패했거나 공백 슬라이스가 반환된 경우
					// 1) 인증을 위한 클라이언트의 호스트 네임이 필요했던 경우, 클라이언트 인증 불가
					// 2) 클라이언트 인증서의 CN 값이나 SAN값을 이용해 IP 주소만을 사용하는 경우라면 처리 가능
					hostnames, err := net.LookupAddr(ip)
					if err != nil {
						t.Errorf("PTR lookup: %v", err)
					}
					hostnames = append(hostnames, ip)
					// 각 검증된 체인 내의 인증서를 순회하며, 중간 인증서 풀을 opts.Intermediates에 할당
					for _, chain := range verifiedChains {
						opts.Intermediates = x509.NewCertPool()
						// leaf certificate 외의 모든 인증서를 중간 인증서 풀에 추가
						for _, cert := range chain[1:] {
							opts.Intermediates.AddCert(cert)
						}

						for _, hostname := range hostnames {
							// 클라이언트 검증 시도
							opts.DNSName = hostname
							_, err = chain[0].Verify(opts)
							if err == nil {
								return nil
							}
						}
					}

					return errors.New("client authentication failed")
				},
			}, nil
		},
	}

	serverAddress := "localhost:44443"
	// 방금 생성한 TLS 구성을 사용해 새로운 TLS 서버 인스턴스를 생성
	server := NewTLSServer(ctx, serverAddress, 0, serverConfig)
	done := make(chan struct{})

	// 별도의 고루틴에서 ListenAndServeTLS 메서드를 호출
	go func() {
		err := server.ListenAndServeTLS("serverCert.pem", "serverKey.pem")
		if err != nil && !strings.Contains(err.Error(),
			"use of closed network connection") {
			t.Error(err)
			return
		}
		done <- struct{}{}
	}()
	// 서버가 연결을 받아들일 준비가 될 때까지 대기
	server.Ready()

	// 클라이언트는 서버의 인증서에서 생성된 새로운 인증서 풀을 받아 옴
	clientPool, err := caCertPool("serverCert.pem")
	if err != nil {
		t.Fatal(err)
	}

	clientCert, err := tls.LoadX509KeyPair("clientCert.pem", "clientKey.pem")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := tls.Dial("tcp", serverAddress, &tls.Config{
		// 클라이언트 내에 서버가 요청하면 응답할 클라이언트 자체의 인증서의 구성을 설정
		Certificates:     []tls.Certificate{clientCert},
		CurvePreferences: []tls.CurveID{tls.CurveP256},
		// 클라이언트는 TLS 구성 내의 RootCA 필드에 존재하는 pool을 사용
		// 즉, 클라이언트는 serverCert.pem 인증서로 서명된 인증서를 사용하는 서버만 신뢰할 것
		RootCAs: clientPool,
	})
	if err != nil {
		t.Fatal(err)
	}

	hello := []byte("hello")
	// 소켓 연결로 읽거나 쓰는 첫 데이터에는 자동으로 클라이언트와 서버 사이에 핸드셰이크 프로세스를 초기화시켜 줌
	_, err = conn.Write(hello)
	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 1024)
	// 서버가 클라이언트의 인증서를 거절하면, 읽기 요청은 인증서 에러가 발생하며 반환될 것
	// 올바르게 인증서를 생성하고 고정하면, 클라이언트와 서버 모두 정상적으로 통신 가능
	n, err := conn.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	if actual := b[:n]; !bytes.Equal(hello, actual) {
		t.Fatalf("expected %q; actual %q", hello, actual)
	}

	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	<-done
}
