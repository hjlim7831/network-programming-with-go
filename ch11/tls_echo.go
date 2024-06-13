package ch11

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func NewTLSServer(ctx context.Context, address string,
	maxIdle time.Duration, tlsConfig *tls.Config) *Server {
	return &Server{
		ctx:       ctx,
		ready:     make(chan struct{}),
		addr:      address,
		maxIdle:   maxIdle,
		tlsConfig: tlsConfig,
	}
}

type Server struct {
	ctx context.Context
	// 서버가 수신 연결 요청을 받아들일 준비가 된 시그널을 처리할 채널
	ready chan struct{}

	addr      string
	maxIdle   time.Duration // 설정값
	tlsConfig *tls.Config   // TLS 구성값
}

// 서버가 수신 연결 요청을 받아들일 준비가 끝날 때까지 블로킹
func (s *Server) Ready() {
	if s.ready != nil {
		<-s.ready
	}
}

// 인증서, 개인키의 전체 경로를 매개변수로 받고, 에러를 반환
func (s *Server) ListenAndServeTLS(certFn, keyFn string) error {
	if s.addr == "" {
		s.addr = "localhost:443"
	}

	// 서버의 주소로 바인딩된 net.Listener 객체를 생성
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("binding to tcp %s: %w", s.addr, err)
	}

	if s.ctx != nil {

		// ctx가 닫히면, 리스너를 종료하는 고루틴을 시작
		go func() {
			<-s.ctx.Done()
			_ = l.Close()
		}()
	}
	// 서버의 ServeTLS 메서드로 생성한 리스너 객체와 인증서 경로, 개인키 경로를 전달함
	return s.ServeTLS(l, certFn, keyFn)
}

func (s Server) ServeTLS(l net.Listener, certFn, keyFn string) error {
	if s.tlsConfig == nil {
		s.tlsConfig = &tls.Config{
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			MinVersion:       tls.VersionTLS12,
			// 구성이 nil이면, 이 값을 true로 설정해 기본 구성을 사용
			// 이 필드는 서버에서 사용하며, 클라이언트가 원하는 암호화 스위트를 기다리지 않고 서버에서 먼저 TLS 협상 단계에서 사용할 암호화 스위트를 사용
			PreferServerCipherSuites: true,
		}
	}

	// 서버의 TLS 구성 값에 최소한 하나의 인증서가 포함되어 있지 않거나, GetCertificate 메서드가 nil을 반환하는 경우
	if len(s.tlsConfig.Certificates) == 0 &&
		s.tlsConfig.GetCertificate == nil {
		// 매개변수로 입력받은 인증서와 개인키의 경로를 사용해 파일시스템으로부터 해당 파일을 읽어 tls.Certificate 객체를 생성
		cert, err := tls.LoadX509KeyPair(certFn, keyFn)
		if err != nil {
			return fmt.Errorf("loading key pair: %v", err)
		}

		s.tlsConfig.Certificates = []tls.Certificate{cert}
	}
	// 이제 서버에는 클라이언트와의 통신에서 사용할 수 있는 최소한 하나 이상의 인증서를 포함한 TLS 구성이 존재함

	// net.Listener 객체와 TLS 구성 정보를 전달해, TLS를 지원하기
	tlsListener := tls.NewListener(l, s.tlsConfig)
	if s.ready != nil {
		close(s.ready)
	}

	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %v", err)
		}

		go func() {
			defer func() { _ = conn.Close() }()

			for {
				if s.maxIdle > 0 {
					err := conn.SetDeadline(time.Now().Add(s.maxIdle))
					if err != nil {
						return
					}
				}

				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					return
				}

				_, err = conn.Write(buf[:n])
				if err != nil {
					return
				}
			}
		}()
	}
}
