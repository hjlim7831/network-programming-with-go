package ch03

import (
	"net"
	"testing"
)

func TestListener(t *testing.T) {
	// net.Listen("네트워크 종류",
	//            "콜론으로 구분된 IP 주소와 포트 문자열")
	// 반환값 : net.Listener 인터페이스, 에러 인터페이스
	// 성공적으로 반환 시 리스너는 특정 IP 주소와 포트 번호에 바인딩됨
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// Close 메서드로 항상 리스너를 우아하게 종료할 것
	// defer를 사용하는 것이 좋음
	// 명시적으로 종료하는 것이 좋은 습관
	defer func() { _ = listener.Close() }()
	// Addr() 메서드로 리스너의 주소 가져오기
	t.Logf("bound to %q", listener.Addr())
}
