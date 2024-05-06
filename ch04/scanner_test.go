package main

import (
	"bufio"
	"net"
	"reflect"
	"testing"
)

const payload = "The bigger the interface, the weaker the abstraction."

func TestScanner(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		_, err = conn.Write([]byte(payload))
		if err != nil {
			t.Error(err)
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// 기본적으로 스캐너는 데이터 스트림으로부터 개행 문자(\n)를 만나면 네트워크 연결로부터 읽은 데이터를 분할
	scanner := bufio.NewScanner(conn)
	// bufio.ScanWords : 공백이나 마침표 등의 단어 경계를 구분하는 구분자를 만날 때마다 데이터를 분할해주는 함수
	scanner.Split(bufio.ScanWords)

	var words []string

	// 네트워크 연결에서 읽을 데이터가 있는 한, 스캐너는 계속해서 데이터를 읽음
	// 한 번 Scan 함수를 호출할 때마다 스캐너는 네트워크 연결로부터 구분자를 찾을 때까지 여러 번의 Read 메서드를 호출하고, 실패할 경우 에러를 반환
	// 이 구현은 네트워크 연결로부터 한 번 이상 데이터를 읽고 구분자를 찾아서 메시지를 반환하는 복잡성을 추상화함
	// scanner는 io.EOF 에러 혹은 그 외의 네트워크 연결로부터 발생하는 에러를 받을 때까지 for 문을 순회하며 데이터를 읽음
	// 네트워크 연결로부터 에러가 발생한 경우, 스캐너의 Err 메서드는 nil이 아닌 에러를 반환할 것
	for scanner.Scan() {
		// scanner.Text() : 네트워크 연결로부터 읽어 들인 데이터 청크를 문자열로 반환
		// 이 경우, 하나의 단어와 근접해 있는 기호가 데이터 청크의 값이 됨
		words = append(words, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		t.Error(err)
	}

	expected := []string{"The", "bigger", "the", "interface,", "the",
		"weaker", "the", "abstraction."}

	if !reflect.DeepEqual(words, expected) {
		t.Fatal("inaccurate scanned word list")
	}
	// 스캐닝된 단어는 go test 커맨드에서 -v 플래그를 실행해 확인 가능
	t.Logf("Scanned words: %#v", words)
}
