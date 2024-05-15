package tftp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

const (
	// 최대 지원하는 데이터그램 크기. 파편화를 피하기 위함
	DatagramSize = 516
	// 데이터 블록의 최대 크기. 4바이트 헤더 크기 제외
	BlockSize = DatagramSize - 4
)

// TFTP 패킷 헤더의 첫 2바이트.
// 작업을 나타내는 OP 코드(operation code)
// 각 OP 코드는 2바이트의 양의 정수로 나타냄
// 서버는 총 4개의 동작을 지원
type OpCode uint16

const (
	// 1) 읽기 요청(Read Request, RPQ)
	OpRRQ OpCode = iota + 1
	// 읽기 전용의 서버를 구현할 것이므로
	// 쓰기 요청(Write Request, WRQ)는 정의 X
	_ // no WRQ support
	// 2) 데이터 작업
	OpData
	// 3) 메시지 승인
	OpAck
	// 4) 에러
	OpErr
)

// 16비트 양의 정수의 에러 코드를 정의
// 서버 기능으로 다운로드만 지원할 것
// -> 모든 에러 코드를 사용하진 않지만
// 클라이언트에서는 메시지 승인 패킷 대신 에러 코드를 반환할 수 있음
type ErrCode uint16

const (
	ErrUnknown ErrCode = iota
	ErrNotFound
	ErrAccessViolation
	ErrDiskFull
	ErrIllegalOp
	ErrUnknownID
	ErrFileExists
	ErrNoUser
)

// 읽기 요청을 나타내는 구조체
// 파일명, 모드 정보를 포함
type ReadReq struct {
	Filename string
	Mode     string
}

// 요청 정보를 슬라이스 바이트로 마샬링할 수 있게 해줌
// 이걸로 서버가 네트워크 연결에 데이터를 쓸 수 있음
// 사실 서버에서는 사용되지 않음 (클라이언트가 사용함)
func (q ReadReq) MarshalBinary() ([]byte, error) {
	mode := "octet"
	if q.Mode != "" {
		mode = q.Mode
	}

	// operation code + filename + 0 byte + mode + 0 byte
	cap := 2 + 2 + len(q.Filename) + 1 + len(mode) + 1

	b := new(bytes.Buffer)
	b.Grow(cap)

	// 패킷을 바이트 슬라이스로 마샬링하기 위해
	// OP 코드를 버퍼에 씀
	err := binary.Write(b, binary.BigEndian, OpRRQ) // write operation code
	if err != nil {
		return nil, err
	}

	_, err = b.WriteString(q.Filename) // write filename
	if err != nil {
		return nil, err
	}
	// null 문자를 버퍼에 씀
	err = b.WriteByte(0) // write 0 byte
	if err != nil {
		return nil, err
	}

	_, err = b.WriteString(mode) // write mode
	if err != nil {
		return nil, err
	}

	err = b.WriteByte(0) // write 0 byte
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (q *ReadReq) UnmarshalBinary(p []byte) error {
	r := bytes.NewBuffer(p)

	var code OpCode

	// 첫 2바이트를 읽고 OP 코드가 읽기 요청인지 확인
	err := binary.Read(r, binary.BigEndian, &code) // read operation code
	if err != nil {
		return err
	}

	if code != OpRRQ {
		return errors.New("invalid RRQ")
	}
	// 첫 널 문자까지 모든 데이터 읽기
	// 이 데이터의 문자열 형태 : 파일명을 나타냄
	q.Filename, err = r.ReadString(0) // read filename
	if err != nil {
		return errors.New("invalid RRQ")
	}

	// 구분자로서의 null 문자 제거
	q.Filename = strings.TrimRight(q.Filename, "\x00") // remove the 0-byte
	if len(q.Filename) == 0 {
		return errors.New("invalid RRQ")
	}

	// 그 다음 널 문자까지 모든 데이터 읽기
	// 이 데이터의 문자열 형태 : 모드 정보
	q.Mode, err = r.ReadString(0) // read mode
	if err != nil {
		return errors.New("invalid RRQ")
	}

	// 구분자로서의 null 문자 제거
	q.Mode = strings.TrimRight(q.Mode, "\x00") // remove the 0-byte
	if len(q.Mode) == 0 {
		return errors.New("invalid RRQ")
	}

	actual := strings.ToLower(q.Mode) // enforce octet mode
	if actual != "octet" {
		return errors.New("only binary transfers supported")
	}
	// 정상적으로 모든 데이터를 읽었다면 nil 반환
	// 이후 서버는 ReadReq 인스턴스를 이용해 클라이언트가 요청한 파일을 읽어옴
	return nil
}

// 데이터 구조체
type Data struct {
	// 현재 블록 번호
	Block uint16
	// 데이터의 원본
	// 바이트 슬라이스 대신 io.Reader 사용
	// -> 페이로드를 어디에서든 얻어올 수 있도록 함
	Payload io.Reader
}

// 파일 시스템에서 파일을 읽으려면? *os.File 객체 사용
// 다른 네트워크 연결로부터 데이터를 읽으려면? net.Conn 객체 사용
// -> io.Reader 인터페이스를 사용하면, 단순하게 바이트 슬라이스를 사용할 때와는 달리 선택의 여지가 생김
// reader는 읽을 수 있는 남은 바이트를 추적해 주며, 많은 코드를 제거할 수 있음

// MarshalBinary 메서드를 이용해 구조체의 값을 수정하려면?
// -> 포인터 리시버를 사용해야 함
// 서버는 reader로부터 모든 데이터를 읽을 때까지
// io.Reader로부터 계속해서 MarshalBinary 메서드를 호출해 일련의 블록 데이터를 읽음
// 클라이언트와 마찬가지로 서버 역시 MarshalBinary 메서드의 반환되는 패킷 크기를 모니터링해야 함
// 패킷의 크기가 516바이트보다 작은 경우, 마지막 패킷이라는 의미가 됨
// -> 서버는 더 이상 MarshalBinary 메서드를 호출하지 않음
func (d *Data) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	b.Grow(DatagramSize)

	// 16비트의 양의 정수인 블록 번호가 언젠가 오버플로가 될 수도 있음
	// 33.5MB (= 65,535 X 512byte)보다 큰 페이로드를 전송하게 되면 블록 번호는 0으로 오버플로 될 것
	// 서버에서는 문제 없이 데이터 패킷을 전송하겠지만, 클라이언트에서는 오버플로를 우아하게 처리하지 못할 수 있음

	// 따라서 TFTP 서버에서 파일을 전송할 때에는
	// 이러한 블록 번호 오버플로가 날 수 있다는 것을 인지하고
	// 1) 클라이언트가 큰 페이로드를 수신할 수 있는지 확인하거나
	// 2) 전혀 다른 프로토콜을 사용하거나
	// 3) 파일 사이즈를 제한해 오버플로를 완화하기
	d.Block++ // block numbers increment from 1

	err := binary.Write(b, binary.BigEndian, OpData) // write operation code
	if err != nil {
		return nil, err
	}

	err = binary.Write(b, binary.BigEndian, d.Block) // write block number
	if err != nil {
		return nil, err
	}

	// write up to BlockSize worth of bytes
	// MarshalBinary 메서드를 호출할 때마다
	// io.CopyN 함수와 BlockSize 상수에 의해 최대 516 바이트를 반환함
	_, err = io.CopyN(b, d.Payload, BlockSize)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return b.Bytes(), nil
}

func (d *Data) UnmarshalBinary(p []byte) error {
	// 데이터 언마샬링을 위해 초기에 데이터 무결성을 확인
	// 1) 기대한 패킷의 크기인지
	// 2) 나머지 바이트들을 읽어도 되는지 확인
	if l := len(p); l < 4 || l > DatagramSize {
		return errors.New("invalid DATA")
	}

	var opcode OpCode

	// OP 코드를 읽고 확인
	err := binary.Read(bytes.NewReader(p[:2]), binary.BigEndian, &opcode)
	if err != nil || opcode != OpData {
		return errors.New("invalid DATA")
	}

	// 블록 번호를 확인
	err = binary.Read(bytes.NewReader(p[2:4]), binary.BigEndian, &d.Block)
	if err != nil {
		return errors.New("invalid DATA")
	}

	// 남은 바이트들을 새로운 버퍼로 집어넣고 Payload 필드에 할당함
	d.Payload = bytes.NewBuffer(p[4:])

	// 클라이언트는 블록 번호를 이용해
	// 1) 서버로 해당하는 번호의 수신 확인 패킷을 보내고
	// 2) 수신된 데이터 블록들의 순서를 올바르게 정렬함
	return nil
}

// 16비트의 양의 정수를 사용해 수신 확인 패킷을 표현
// 이 정수는 수신 확인된 블록 번호를 나타냄
type Ack uint16

// OP 코드와 블록 번호 -> 바이트 슬라이스로 마샬링
func (a Ack) MarshalBinary() ([]byte, error) {
	cap := 2 + 2 // operation code + block number

	b := new(bytes.Buffer)
	b.Grow(cap)

	err := binary.Write(b, binary.BigEndian, OpAck) // write operation code
	if err != nil {
		return nil, err
	}

	err = binary.Write(b, binary.BigEndian, a) // write block number
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// 바이트 -> Ack 객체로 언마샬링
func (a *Ack) UnmarshalBinary(p []byte) error {
	var code OpCode

	r := bytes.NewReader(p)

	err := binary.Read(r, binary.BigEndian, &code) // read operation code
	if err != nil {
		return err
	}

	if code != OpAck {
		return errors.New("invalid ACK")
	}

	return binary.Read(r, binary.BigEndian, a) // read block number
}

// 에러 타입 : 에러 패킷을 생성하는 데 필요한 최소 데이터를 포함함
type Err struct {
	Error   ErrCode
	Message string
}

// 바이트 버퍼를 생성해 반환
func (e Err) MarshalBinary() ([]byte, error) {
	// operation code + error code + message + 0 byte
	cap := 2 + 2 + len(e.Message) + 1

	b := new(bytes.Buffer)
	b.Grow(cap)

	err := binary.Write(b, binary.BigEndian, OpErr) // write operation code
	if err != nil {
		return nil, err
	}

	err = binary.Write(b, binary.BigEndian, e.Error) // write error code
	if err != nil {
		return nil, err
	}

	_, err = b.WriteString(e.Message) // write message
	if err != nil {
		return nil, err
	}

	err = b.WriteByte(0) // write 0 byte
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (e *Err) UnmarshalBinary(p []byte) error {
	r := bytes.NewBuffer(p)

	var code OpCode

	// OP 코드 검증
	err := binary.Read(r, binary.BigEndian, &code) // read operation code
	if err != nil {
		return err
	}

	if code != OpErr {
		return errors.New("invalid ERROR")
	}
	// 에러 코드 읽기
	err = binary.Read(r, binary.BigEndian, &e.Error) // read error code
	if err != nil {
		return err
	}
	// 에러 메시지 읽기
	e.Message, err = r.ReadString(0) // read error message
	// null 문자 떼내기
	e.Message = strings.TrimRight(e.Message, "\x00") // remove the 0-byte

	return err
}
