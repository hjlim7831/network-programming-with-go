package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// 정의할 메시지 타입을 나타내는 상수
const (
	// BinaryType, StringType 을 생성
	// 각 타입의 세부 구현 정보를 요약한 후, 필요에 맞게 타입을 생성
	BinaryType uint8 = iota + 1
	StringType
	// 보안상의 문제로 인해 최대 페이로드 크기를 반드시 정의해 주어야 함
	MaxPayloadSize uint32 = 10 << 20 // 10 MB
)

var ErrMaxPayloadSize = errors.New("maximum payload size exceeded")

// 각 타입별 메시지들이 구현해야 하는 Payload라는 이름의 인터페이스를 정의
// io.ReaderFrom, io.WriterTo : 각 타입별 메시지를 reader 로부터 읽을 수 있고 writer에 쓸 수 있게 해주는 기능의 형태를 제공
// 세부 구현은 필요에 맞게 하면 됨
// Payload 인터페이스를 encoding.BinaryMarshaler 인터페이스를 구현하도록 만들면, Payload를 구현할 타입별 메시지가 스스로 바이트 슬라이스로 마샬링될 수 있게 할 수 있음
// Payload 인터페이스를 encoding.BinaryUnmarshaler 인터페이스를 구현하도록 만들면, 바이트 슬라이스로부터 메시지를 언마샬링 되게 할 수 있음
// 하지만 네트워크 연결 수준에서는 바이트 슬라이스를 그대로 다루지 않으므로, Payload 인터페이스를 그대로 사용 (다음 장부터 바이너리 인코딩 인터페이스를 사용함)
type Payload interface {
	fmt.Stringer
	io.ReaderFrom
	io.WriterTo
	Bytes() []byte
}

// 바이트 슬라이스
type Binary []byte

// Bytes 메서드는 자기 자신을 반환
func (m Binary) Bytes() []byte { return m }

// String 메서드는 자기 자신을 문자열로 캐스팅하여 반환
func (m Binary) String() string { return string(m) }

// WriteTo 메서드는 io.Writer 인터페이스를 매개변수로 받아서 writer에 쓰인 바이트 수와 에러 인터페이스를 반환
func (m Binary) WriteTo(w io.Writer) (int64, error) {
	// WriteTo 메서드는 1바이트의 "타입(T)"을 writer에 씀 (이 경우 BinaryType)
	err := binary.Write(w, binary.BigEndian, BinaryType) // 1-byte type
	if err != nil {
		return 0, err
	}
	var n int64 = 1
	// Binary 인스턴스의 "길이(L)"인 4바이트를 writer에 씀
	err = binary.Write(w, binary.BigEndian, uint32(len(m))) // 4-byte size
	if err != nil {
		return n, err
	}
	n += 4
	// Binary 인스턴스 자체의 "값(V)"을 writer에 씀
	o, err := w.Write(m) // 페이로드

	return n + int64(o), err
}

func (m *Binary) ReadFrom(r io.Reader) (int64, error) {
	var typ uint8
	// reader로부터 1바이트를 typ 변수에 읽어 들인 후, 타입이 BinaryType인지 확인
	err := binary.Read(r, binary.BigEndian, &typ) // 1-byte type
	if err != nil {
		return 0, err
	}
	var n int64 = 1
	if typ != BinaryType {
		return n, errors.New("invalid Binary")
	}
	// size 변수에 다음 4바이트를 읽어 들임
	var size uint32
	err = binary.Read(r, binary.BigEndian, &size) // 4-byte size
	if err != nil {
		return n, err
	}
	n += 4
	// 최대 페이로드 크기가 지정되어 있음
	// 페이로드 크기: 최대 4GB
	// 이렇게 큰 페이로드를 처리하게 되면, 악의적인 의도를 가진 사용자가 서비스 거부 공격을 시도하여 서버상의 가용 가능한 RAM을 전부 소비해 버리기 쉬움
	// 최대 페이로드 크기를 합리적인 크기로 관리하면, 서비스 거부 등의 악의적인 사용자로부터 메모리 소비를 방지할 수 있음
	if size > MaxPayloadSize {
		return n, ErrMaxPayloadSize
	}
	// size 변숫값을 Binary 인스턴스의 크기로 새로운 바이트 슬라이스를 할당
	*m = make([]byte, size)
	// Binary 인스턴스의 바이트 슬라이스를 읽음
	o, err := r.Read(*m) // payload

	return n + int64(o), err
}

type String string

// Bytes 메서드 : 자기 자신의 String 인스턴스 값을 바이트 슬라이스로 형 변환
// String 메서드 : 자기 자신의 String 인스턴스 값을 베이스 타입인 string으로 형 변환
func (m String) Bytes() []byte  { return []byte(m) }
func (m String) String() string { return string(m) }

// String 타입의 WriteTo 메서드
func (m String) WriteTo(w io.Writer) (int64, error) {
	// 1 바이트의 Type 작성 : SringType
	err := binary.Write(w, binary.BigEndian, StringType) // 1-byte type
	if err != nil {
		return 0, err
	}
	var n int64 = 1
	// 4 바이트의 size 작성
	err = binary.Write(w, binary.BigEndian, uint32(len(m))) // 4-byte size
	if err != nil {
		return n, err
	}
	n += 4

	// String 인스턴스의 값을 writer로 쓰기 전 바이트 슬라이스로 형변환
	o, err := w.Write([]byte(m)) // 페이로드

	return n + int64(o), err
}

func (m *String) ReadFrom(r io.Reader) (int64, error) {
	var typ uint8
	err := binary.Read(r, binary.BigEndian, &typ) // 1-byte type
	if err != nil {
		return 0, err
	}
	// typ 변수를 StringType과 비교
	var n int64 = 1
	if typ != StringType {
		return n, errors.New("invalid String")
	}

	var size uint32
	err = binary.Read(r, binary.BigEndian, &size) // 4-byte size
	if err != nil {
		return n, err
	}
	n += 4
	if size > MaxPayloadSize {
		return n, ErrMaxPayloadSize
	}

	buf := make([]byte, size)
	o, err := r.Read(buf) // payload
	if err != nil {
		return n, err
	}
	// reader로부터 읽은 값을 String으로 형변환
	*m = String(buf)

	return n + int64(o), nil
}

// decode 함수 : io.Reader 인터페이스를 매개변수로 받아, Payload 인터페이스와 error 인터페이스를 반환
func decode(r io.Reader) (Payload, error) {
	var typ uint8
	// 타입 추론을 위해 먼저 reader로부터 1 바이트를 읽어 들임
	err := binary.Read(r, binary.BigEndian, &typ)
	if err != nil {
		return nil, err
	}
	// payload 변수를 생성해 디코딩된 타입의 값을 저장
	var payload Payload

	// 읽어 들인 타입이 미리 정의한 상수 타입이면, payload 변수에 해당 상수 타입을 할당
	switch typ {
	case BinaryType:
		payload = new(Binary)
	case StringType:
		payload = new(String)
	// reader로부터 읽은 바이트를 Binary 타입이나 String 타입으로 디코딩할 수 없는 경우, nil 값의 Payload와 함께 에러를 반환
	default:
		return nil, errors.New("unknown type")
	}
	// 이미 타입을 추론하기 위해 1바이트를 읽어 들였으므로, MultiReader를 사용.
	// 이미 읽은 바이트(typ)를 다음에 읽을 바이트와 연결하는 데 사용

	// 이렇게 사용하는 것이 최적은 아님.
	// ReadFrom 메서드로부터 타입을 알아내기 위해 첫 1바이트를 읽어야만 하는 동작을 제거하는 것이 더 적절한 리팩토링 방법임
	_, err = payload.ReadFrom(
		io.MultiReader(bytes.NewReader([]byte{typ}), r))
	if err != nil {
		return nil, err
	}

	return payload, nil
}
