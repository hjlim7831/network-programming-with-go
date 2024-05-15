package tftp

import (
	"bytes"
	"errors"
	"log"
	"net"
	"time"
)

// Server represents a read-only TFTP server that supports a subset of
// RFC 1350.
type Server struct {
	Payload []byte        // 모든 읽기 요청에 반환된 페이로드
	Retries uint8         // 전송 실패 시 재시도 횟수
	Timeout time.Duration // 전송 승인을 기다릴 기간 (타임아웃 기간)
}

func (s Server) ListenAndServe(addr string) error {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	log.Printf("Listening on %s ...\n", conn.LocalAddr())

	return s.Serve(conn)
}

// net.PacketConn 객체를 매개변수로 받고, 해당 객체를 이용해 읽기 수신 요청에 활용
// 네트워크 연결을 닫으면? 메서드가 반환될 것
func (s *Server) Serve(conn net.PacketConn) error {
	if conn == nil {
		return errors.New("nil connection")
	}

	if s.Payload == nil {
		return errors.New("payload is required")
	}

	if s.Retries == 0 {
		s.Retries = 10
	}

	if s.Timeout == 0 {
		s.Timeout = 6 * time.Second
	}

	var rrq ReadReq

	for {
		buf := make([]byte, DatagramSize)

		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}
		// 서버는 네트워크 연결로부터 516 바이트의 데이터를 읽고, ReadReq 객체로 언마샬링을 시도
		err = rrq.UnmarshalBinary(buf)
		if err != nil {
			log.Printf("[%s] bad request: %v", addr, err)
			continue
		}
		// 네트워크 연결에서 읽은 데이터가 읽기 요청인 경우, 서버는 데이터를 고루틴의 핸들러로 전달
		go s.handle(addr.String(), rrq)
	}
}

// 클라이언트 주소와 읽기 요청을 매개변수로 받는 Server 타입의 메서드
// Server의 필드 값에 접근해야 할 필요성이 있으므로, 함수가 아닌 메서드로 정의됨
func (s Server) handle(clientAddr string, rrq ReadReq) {
	log.Printf("[%s] requested file: %s", clientAddr, rrq.Filename)
	// 클라이언트와 연결 맺기
	// conn 객체는 클라이언트로부터 Read 함수 호출마다 송신자의 주소를 확인할 필요 없이 읽기 전용 모드로 패킷을 수신할 수 있음
	conn, err := net.Dial("udp", clientAddr)
	if err != nil {
		log.Printf("[%s] dial: %v", clientAddr, err)
		return
	}

	defer func() { _ = conn.Close() }()

	var (
		ackPkt Ack
		errPkt Err
		// 서버의 페이로드를 사용해 데이터 객체 준비
		dataPkt = Data{Payload: bytes.NewReader(s.Payload)}
		buf     = make([]byte, DatagramSize)
	)

NEXTPACKET:
	// for문에서 각 데이터 패킷을 전송
	// 이 for문은 데이터 패킷의 크기가 516 바이트(DatagramSize)인 경우 계속해서 데이터를 전송
	for n := DatagramSize; n == DatagramSize; {
		// 데이터 객체를 바이트 슬라이스로 마샬링한 후
		data, err := dataPkt.MarshalBinary()
		if err != nil {
			log.Printf("[%s] preparing data packet: %v", clientAddr, err)
			return
		}

	RETRY:
		// 재시도 횟수 만큼 or 성공적으로 전송할 때까지 데이터 패킷을 재전송하기 위한 for 문을 순회
		for i := s.Retries; i > 0; i-- {
			// n 의 값을 쓰인 데이터 패킷의 바이트 수로 업데이트
			n, err = conn.Write(data) // send the data packet
			if err != nil {
				log.Printf("[%s] write: %v", clientAddr, err)
				return
			}
			// 전송 완료를 결정하기 전, 클라이언트가 마지막 데이터 패킷을 성공적으로 수신했는지 확인해야 함
			// 1) 클라이언트로부터 바이트를 읽은 후
			// wait for the client's ACK packet
			_ = conn.SetReadDeadline(time.Now().Add(s.Timeout))

			_, err = conn.Read(buf)
			if err != nil {
				if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
					continue RETRY
				}

				log.Printf("[%s] waiting for ACK: %v", clientAddr, err)
				return
			}
			// 2) Ack 객체나 Err 객체로 언마샬링을 시도
			switch {
			// Ack 객체로 언마샬링 되면?
			case ackPkt.UnmarshalBinary(buf) == nil:
				// 객체의 Block 값을 확인해 현재 데이터 패킷에 해당하는 블록 번호를 확인할 수 있음
				// 블록 번호가 맞으면? NEXTPACKET 다시 순회
				// 맞지 않으면? 현재의 데이터 패킷을 재전송
				if uint16(ackPkt) == dataPkt.Block {
					// received ACK; send next data packet
					continue NEXTPACKET
				}
			// Err 객체로 언마샬링 되면?
			case errPkt.UnmarshalBinary(buf) == nil:
				// 클라이언트가 에러를 반환했음을 알 수 있음
				// 해당 사실을 로깅하고, 일찍이 함수를 반환
				// 전체 페이로드를 보내기 전에 전송이 종료되었음을 의미함
				// 이 경우, 복구가 불가능하므로 클라이언트는 파일을 다시 요청해야만 함
				log.Printf("[%s] received error: %v",
					clientAddr, errPkt.Message)
				return
			default:
				log.Printf("[%s] bad packet", clientAddr)
			}
		}

		log.Printf("[%s] exhausted retries", clientAddr)
		return
	}

	log.Printf("[%s] sent %d blocks", clientAddr, dataPkt.Block)
}
