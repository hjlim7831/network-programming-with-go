package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"

	"github.com/awoodbeck/gnp/ch07/creds/auth"
)

func init() {
	flag.Usage = func() {
		// 매개변수로 그룹 이름이 들어오기를 원함
		// 허용된 그룹 이름 목록 맵에 각 그룹 ID에 해당하는 이름을 추가할 것
		_, _ = fmt.Fprintf(flag.CommandLine.Output(),
			"Usage:\n\t%s <group names>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

// 그룹 이름을 포함하는 문자열 슬라이스를 매개변수로 받아
// 각 이름에 해당하는 그룹 정보를 조회하고, 각 그룹 ID를 그룹 맵에 주입
func parseGroupNames(args []string) map[string]struct{} {
	groups := make(map[string]struct{})

	for _, arg := range args {
		grp, err := user.LookupGroup(arg)
		if err != nil {
			log.Println(err)
			continue
		}

		groups[grp.Gid] = struct{}{}
	}

	return groups
}

func main() {
	flag.Parse()
	// 커맨드 라인 매개변수를 파싱해, 허용된 그룹 ID의 맵 생성
	groups := parseGroupNames(flag.Args())
	// /tmp/creds.sock 소켓 리스너 생성
	socket := filepath.Join(os.TempDir(), "creds.sock")
	addr, err := net.ResolveUnixAddr("unix", socket)
	if err != nil {
		log.Fatal(err)
	}

	s, err := net.ListenUnix("unix", addr)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	// 인터럽트 시그널로 서비스를 갑작스레 종료시키면
	// net.ListenUnix 함수를 사용하였음에도 불구하고 Go 가 소켓 파일을 정리하고 제거하지 못하게 됨
	// 먼저 시그널을 대기하고
	signal.Notify(c, os.Interrupt)
	// 별도의 고루틴에서 해당 시그널을 받은 후, 리스너를 종료하도록 함
	// 이렇게 하면, Go에서 적절하게 소켓 파일을 처리할 수 있음
	go func() {
		<-c
		_ = s.Close()
	}()

	fmt.Printf("Listening on %s ...\n", socket)

	for {
		// 리스너는 AcceptUnix 메서드를 이용해 연결 수립 요청을 받아들임
		conn, err := s.AcceptUnix()
		if err != nil {
			break
		}
		// peer의 인증 정보가 허용되었는지 확인
		if auth.Allowed(conn, groups) {
			_, err = conn.Write([]byte("Welcome\n"))
			// 허용된 peer는 계속 연결 유지
			if err == nil {
				// handle the connection in a goroutine here
				continue
			}
		}
		// 허용되지 않은 peer는 즉시 연결 종료
		_, err = conn.Write([]byte("Access denied\n"))
		if err != nil {
			log.Println(err)
		}

		_ = conn.Close()
	}
}
