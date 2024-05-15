package main

import (
	"crypto/sha512"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s file...\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	// 커맨드 라인 매개변수로 하나 이상의 파일 경로를 받음
	for _, file := range flag.Args() {
		fmt.Printf("\n%s =>\n%s\n", file, checksum(file))
	}
}

func checksum(file string) string {
	// 그 파일 내용을 가져와서
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err.Error()
	}
	// 체크섬을 생성
	return fmt.Sprintf("%x", sha512.Sum512_256(b))
}
