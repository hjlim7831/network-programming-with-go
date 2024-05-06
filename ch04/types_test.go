package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"reflect"
	"testing"
)

func TestPayloads(t *testing.T) {
	b1 := Binary("Clear is better than clever.")
	b2 := Binary("Don't panic.")
	s1 := String("Errors are values.")
	payloads := []Payload{&b1, &s1, &b2}

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
		// 각 타입을 페이로드 슬라이스 형태로 전송
		for _, p := range payloads {
			_, err = p.WriteTo(conn)
			if err != nil {
				t.Error(err)
				break
			}
		}
	}()
	// 리스너로의 연결을 수립
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	for i := 0; i < len(payloads); i++ {
		// 각 페이로드를 디코딩
		actual, err := decode(conn)
		if err != nil {
			t.Fatal(err)
		}
		// 디코딩된 타입을 서버가 전송한 타입과 비교
		// expected : 처음에 전송하기 전 정의한 payload
		// actual : conn으로 받은 뒤 decode된 값
		if expected := payloads[i]; !reflect.DeepEqual(expected, actual) {
			t.Errorf("value mismatch: %v != %v", expected, actual)
			continue
		}

		t.Logf("[%T] %[1]q", actual)
	}
}

func TestMaxPayloadSize(t *testing.T) {
	// buffer 생성
	buf := new(bytes.Buffer)
	// BinaryType (T) 를 buffer에 작성
	err := buf.WriteByte(BinaryType)
	if err != nil {
		t.Fatal(err)
	}
	// size(L)로 1GB의 크기를 buf에 작성
	err = binary.Write(buf, binary.BigEndian, uint32(1<<30))
	if err != nil {
		t.Fatal(err)
	}

	var b Binary
	// 이 버퍼가 Binary 타입의 ReadFrom 메서드로 넘겨지면
	_, err = b.ReadFrom(buf)
	// ErrMaxPayloadSize 에러를 반환받음
	if err != ErrMaxPayloadSize {
		t.Fatalf("expected ErrMaxPayloadSize; actual: %v", err)
	}
}
