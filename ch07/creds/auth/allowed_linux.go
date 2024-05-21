package auth

import (
	"fmt"
	"log"
	"net"
	"os/user"

	"golang.org/x/sys/unix"
)

func Allowed(conn *net.UnixConn, groups map[string]struct{}) bool {
	if conn == nil || groups == nil || len(groups) == 0 {
		return false
	}

	// peer의 유닉스 인증 정보를 가져오기 위해, 먼저 net.UnixConn 파일 객체를 변수에 저장
	// 파일 객체 : 호스트상의 유닉스 도메인 소켓 연결 객체
	// net.TCPConn 객체와도 유사함

	// 연결 객체로부터 파일 디스크립터 정보를 획득해야 하므로, 리스너의 Accept 메서드로부터 반환된 net.UnixConn 객체의 포인터를 Allowed 함수의 매개변수로 넘겨줘야 함
	file, _ := conn.File()
	defer func() { _ = file.Close() }()

	var (
		err   error
		ucred *unix.Ucred
	)

	for {
		// 파일 객체의 디스크립터(file.Fd()), 어느 프로토콜 계층에 속하였는지를 나타내는 상수 (unix.SOL_SOCKET), 옵션 값 (unix.SO_PEERCRED)를 넘겨줌
		// 리눅스 커널에서 소켓 옵션 값을 얻어 오려면, 해당하는 옵션과 해당 옵션이 존재하는 계층 값이 모두 필요함
		// unix.SOL_SOCKET : 리눅스 커널에서 소켓 계층의 옵션 값이 필요함을 알려줌
		// unix.SO_PEERCRED : 리눅스 커널에 피어의 인증 정보가 필요하다고 알려줌
		ucred, err = unix.GetsockoptUcred(int(file.Fd()), unix.SOL_SOCKET,
			unix.SO_PEERCRED)
		// 리눅스 커널이 유닉스 도메인 소켓 계층의 피어 인증 정보를 찾으면, 위 함수는 정상적인 unix.Ucred 객체의 포인터를 반환함
		if err == unix.EINTR {
			continue // syscall interrupted, try again
		}
		if err != nil {
			log.Println(err)
			return false
		}

		break
	}
	// unix.Ucred 객체에는
	// 1) peer의 프로세스 정보
	// 2) 사용자 ID, 그룹 ID 정보 가 있음
	// peer의 사용자 ID 정보를 user.LookupId 함수에 매개변수로 전달
	u, err := user.LookupId(fmt.Sprint(ucred.Uid))
	if err != nil {
		log.Println(err)
		return false
	}
	// 함수가 성공적으로 호출되면, 사용자 객체로부터 그룹 ID의 목록을 반환
	// 사용자가 하나 이상의 그룹에 속할 수 있으므로, 각각의 그룹에 대한 권한을 확인해 봐야 함
	gids, err := u.GroupIds()
	if err != nil {
		log.Println(err)
		return false
	}
	// 허용된 그룹들에 대해 각각의 그룹 ID를 비교
	// peer의 그룹 ID 중 하나가 허용된 그룹과 일치한다면, true를 반환해 peer가 연결할 수 있도록 함
	for _, gid := range gids {
		if _, ok := groups[gid]; ok {
			return true
		}
	}

	return false
}
