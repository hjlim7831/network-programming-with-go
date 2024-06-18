package json

import (
	"encoding/json"
	"io"

	"github.com/awoodbeck/gnp/ch12/housework"
)

func Load(r io.Reader) ([]*housework.Chore, error) {
	var chores []*housework.Chore

	// json.NewDecoder 함수에 매개변수로 io.Reader 인터페이스를 받은 뒤, 디코더를 반환
	// 집안일 슬라이스의 포인터를 매개변수로 전달해, 디코더의 Decode 메서드를 호출
	// 디코더는 io.Reader로부터 JSON 데이터를 읽어서 역직렬화한 뒤, 집안일 슬라이스를 만들어 냄
	return chores, json.NewDecoder(r).Decode(&chores)
}

// io.Writer와 집안일 슬라이스를 매개변수로 받음
func Flush(w io.Writer, chores []*housework.Chore) error {
	// json.NewEncoder 함수에 매개변수로 io.Writer를 전달
	// JSON 데이터를 직렬화한 뒤 해당 데이터를 io.Writer로 씀
	return json.NewEncoder(w).Encode(chores)
}
