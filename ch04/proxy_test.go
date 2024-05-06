package main

import (
	"io"
	"net"
	"sync"
	"testing"
)

// net.Conn 인터페이스 대신 범용적인 io.Reader 인터페이스와 io.Writer 인터페이스를 매개변수로 받음
// 더 활용 범위가 넓음
// 이를 사용해 데이터를 네트워크 연결로부터 os.Stdout, *bytes.Buffer, *os.File 외에 io.Writer 인터페이스를 구현한 많은 객체들로 데이터를 프락시할 수 있음
// io.Reader 인터페이스를 구현한 임의의 객체로부터 데이터를 읽어서 writer로 전송할 수 있음

// 이 프락시 구현은 from reader가 io.Writer 인터페이스를 구현하고, to writer가 io.Reader 인터페이스를 구현하였다면 서로의 요청에 응답할 수도 있음
func proxy(from io.Reader, to io.Writer) error {
	// from 이 io.Writer 인터페이스를 구현하였는지 확인
	fromWriter, fromIsWriter := from.(io.Writer)
	// to 가 io.Reader 인터페이스를 구현하였는지 확인
	toReader, toIsReader := to.(io.Reader)

	if toIsReader && fromIsWriter {
		// Send replies since "from" and "to" implement the
		// necessary interfaces.
		go func() { _, _ = io.Copy(fromWriter, toReader) }()
	}

	_, err := io.Copy(to, from)
	return err
}

func TestProxy(t *testing.T) {
	var wg sync.WaitGroup

	// 서버는 "ping" 메시지를 듣고 "pong" 메시지를 응답
	// 다른 모든 메시지들은 그대로 client로 돌려줌
	// 연결 요청을 수신할 수 있는 서버를 초기화함
	server, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()

				for {
					buf := make([]byte, 1024)
					n, err := c.Read(buf)
					if err != nil {
						if err != io.EOF {
							t.Error(err)
						}

						return
					}

					switch msg := string(buf[:n]); msg {
					case "ping":
						_, err = c.Write([]byte("pong"))
					default:
						_, err = c.Write(buf[:n])
					}

					if err != nil {
						if err != io.EOF {
							t.Error(err)
						}

						return
					}
				}
			}(conn)
		}
	}()
	// proxyServer는 메시지를 클라이언트 연결로부터 destinationServer로 프락시함
	// destinationServer 서버에서 온 응답 메시지는 역으로 클라이언트에게 프락시됨
	// 클라이언트와 목적지 서버 간의 메시지 전달을 처리해 주는 프락시 서버를 셋업
	proxyServer, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			// 프락시 서버는 클라이언트의 연결 요청을 수신함
			// 클라이언트와의 연결이 수립되면
			conn, err := proxyServer.Accept()
			if err != nil {
				return
			}

			go func(from net.Conn) {
				defer from.Close()
				// proxy 함수는 목적지 서버와의 연결을 수립함
				to, err := net.Dial("tcp",
					server.Addr().String())
				if err != nil {
					t.Error(err)
					return
				}
				defer to.Close()
				// 메시지를 프락싱함
				// 프락시 서버에 매개변수로 프락싱할 두 개의 net.Conn 객체를 전달
				// net.Conn 인터페이스는 io.ReadWriter 인터페이스를 구현하기 때문에 서버의 프락시는 서로 응답할 수 있음
				// 이후 io.Copy 함수는 출발지 노드 혹은 목적지 노드로부터 net.Conn 객체에서 Read 메서드로 읽은 모든 데이터를 목적지 노드 혹은 출발지 노드를 향해 Write 메서드로 씀
				err = proxy(from, to)
				if err != nil && err != io.EOF {
					t.Error(err)
				}
			}(conn)
		}
	}()

	conn, err := net.Dial("tcp", proxyServer.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// 일련의 테스트 간에 프락시를 수행해
	// 1) ping 메시지가 pong으로 응답되는지
	// 2) 그 외의 모든 메시지가 그대로 반환되어 에코 서버가 제대로 동작하는지
	// 확인
	msgs := []struct{ Message, Reply string }{
		{"ping", "pong"},
		{"pong", "pong"},
		{"echo", "echo"},
		{"ping", "pong"},
	}

	for i, m := range msgs {
		_, err = conn.Write([]byte(m.Message))
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if actual := string(buf[:n]); actual != m.Reply {
			t.Errorf("%d: expected reply: %q; actual: %q",
				i, m.Reply, actual)
		}
	}

	_ = conn.Close()
	_ = proxyServer.Close()
	_ = server.Close()
	wg.Wait()
}
