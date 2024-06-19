package protobuf

import (
	"io"

	"google.golang.org/protobuf/proto"

	// protoc 컴파일러에서 생성된 패키지 명인 v1을 임포트
	"github.com/awoodbeck/gnp/ch12/housework/v1"
)

func Load(r io.Reader) ([]*housework.Chore, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var chores housework.Chores

	return chores.Chores, proto.Unmarshal(b, &chores)
}

// 컴파일되어 생성된 Chores 타입은 Chore 슬라이스의 포인터를 받는 Chores 필드를 갖는 구조체
// Go의 프로토콜 버퍼 패키지는 인코더와 디코더를 구현하지 않음
// 객체를 바이트로 마샬링한 뒤 마샬링된 바이트를 io.Writer로 쓰고, io.Reader로부터 읽은 바이트를 언마샬링하는 코드를 직접 구현해야 함
func Flush(w io.Writer, chores []*housework.Chore) error {
	b, err := proto.Marshal(&housework.Chores{Chores: chores})
	if err != nil {
		return err
	}

	_, err = w.Write(b)

	return err
}
