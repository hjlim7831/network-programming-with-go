package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/awoodbeck/gnp/ch12/housework"
	// 직렬화 포맷 임포트
	storage "github.com/awoodbeck/gnp/ch12/json"
	// storage "github.com/awoodbeck/gnp/ch12/gob"
	// storage "github.com/awoodbeck/gnp/ch12/protobuf"
)

var dataFile string

func init() {
	flag.StringVar(&dataFile, "file", "housework.db", "data file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			// 커맨드 라인의 매개변수와 사용법을 나타냄
			`Usage: %s [flags] [add chore, ...|complete #]
    add         add comma-separated chores
    complete    complete designated chore

Flags:
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

// housework.Chore 구조체 인스턴스의 포인터 슬라이스를 반환
func load() ([]*housework.Chore, error) {
	// 데이터 파일이 존재하지 않을 경우, 공백 슬라이스를 반환하며 종료됨
	// 이는 애플리케이션을 최초로 실행했을 때 발생
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		return make([]*housework.Chore, 0), nil
	}
	// 애플리케이션이 데이터 파일을 읽은 경우, 해당 파일을 열고
	df, err := os.Open(dataFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := df.Close(); err != nil {
			fmt.Printf("closing data file: %v", err)
		}
	}()
	// io.Reader를 매개변수로 받는 스토리지의 Load 함수로 전달
	return storage.Load(df)
}

func flush(chores []*housework.Chore) error {
	// 새로운 파일을 만들거나 이미 존재하는 파일을 버리고
	df, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := df.Close(); err != nil {
			fmt.Printf("closing data file: %v", err)
		}
	}()

	// 해당 파일 포인터와 함께 집안일 슬라이스를 스토리지의 Flush 함수에 전달
	return storage.Flush(df, chores)
}

func list() error {
	// 스토리지로부터 집안일 목록을 로드
	chores, err := load()
	if err != nil {
		return err
	}

	// 로드된 목록에 아무런 집안일이 없다면, 없는대로 표준 출력에 출력
	if len(chores) == 0 {
		fmt.Println("You're all caught up!")
		return nil
	}
	// 집안일이 있으면, 아래처럼 헤더와 내용을 출력
	fmt.Println("#\t[X]\tDescription")
	for i, chore := range chores {
		c := " "
		if chore.Complete {
			c = "X"
		}
		fmt.Printf("%d\t[%s]\t%s\n", i+1, c, chore.Description)
	}

	return nil
}

func add(s string) error {
	// 1) 집안일 목록을 스토리지에서 가져온 후
	chores, err := load()
	if err != nil {
		return err
	}
	// 2) 수정한 뒤
	// 동시에 하나 이상의 집안일을 목록에 추가하기 위해, 여러 집안일의 설명을 쉼표로 구분된 문자열로 받음
	for _, chore := range strings.Split(s, ",") {
		if desc := strings.TrimSpace(chore); desc != "" {
			chores = append(chores, &housework.Chore{
				Description: desc,
			})
		}
	}
	// 3) 변화된 부분을 다시 스토리지에 저장
	return flush(chores)
}

// 매개변수로 완료하고자 하는 집안일을 커맨드 라인에서 받은 후, 정수로 변환함
func complete(s string) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	chores, err := load()
	if err != nil {
		return err
	}

	if i < 1 || i > len(chores) {
		return fmt.Errorf("chore %d not found", i)
	}
	// 집안일의 목록 번호로 0이 아닌 1부터 시작함
	// 슬라이스에 맞춰, 1을 빼주어야 함
	chores[i-1].Complete = true
	// 변경된 부분을 스토리지에 저장
	return flush(chores)
}

func main() {
	flag.Parse()

	var err error

	switch strings.ToLower(flag.Arg(0)) {
	case "add":
		err = add(strings.Join(flag.Args()[1:], " "))
	case "complete":
		err = complete(flag.Arg(1))
	}

	if err != nil {
		log.Fatal(err)
	}

	err = list()
	if err != nil {
		log.Fatal(err)
	}
}
