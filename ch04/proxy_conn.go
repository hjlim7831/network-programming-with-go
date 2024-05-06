package main

import (
	"io"
	"net"
)

// 두 노드가 서로 직접 연결한 것처럼 프락시로 데이터를 주고받을 수 있음
func proxyConn(source, destination string) error {
	// 출발지 노드와 연결을 생성
	connSource, err := net.Dial("tcp", source)
	if err != nil {
		return err
	}
	defer connSource.Close()
	// 목적지 노드와 연결을 생성
	connDestination, err := net.Dial("tcp", destination)
	if err != nil {
		return err
	}
	defer connDestination.Close()
	// io.Copy 함수가 데이터 입출력에서 직접 처리하기 어려운 모든 부분을 처리해 줌
	// io.Copy(io.Writer, io.Reader) : reader로부터 읽는 모든 데이터를 writer로 씀
	// reader가 io.EOF를 반환하거나, reader 혹은 writer가 error를 반환할 경우 함수는 종료됨
	// io.Copy 함수 동작 도중, reader로부터 모든 데이터를 읽었다는 의미의 io.EOF 에러 이외의 에러가 발생했을 때만 error를 반환함

	// connDestination 에서 데이터를 읽어, connSource 로 데이터를 씀
	// 두 노드 중 하나의 연결이 끊어지면 io.Copy는 자동으로 종료되므로 이 고루틴이 메모리 누수를 일으킬 걱정은 하지 않아도 됨
	// connSource <- connDestination (replies)
	go func() { _, _ = io.Copy(connSource, connDestination) }()

	// 각 연결의 Close 메서드가 호출되어 io.Copy 함수가 반환되면 고루틴이 종료됨
	// connDestination <- connSource
	_, err = io.Copy(connDestination, connSource)

	return err
}

var _ = proxyConn
