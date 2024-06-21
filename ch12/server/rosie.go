package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/awoodbeck/gnp/ch12/housework/v1"
)

type Rosie struct {
	// 하나 이상의 클라이언트가 동시에 서비스를 사용할 수 있도록, 뮤텍스를 사용해 동시 접근을 보호
	mu sync.Mutex
	// 메모리상에 집안일 목록을 저장
	chores []*housework.Chore
}

// Add, Complete, List 메서드는 모두 클라이언트에게 전달되는 응답 메시지 혹은 에러를 반환

func (r *Rosie) Add(_ context.Context, chores *housework.Chores) (
	*housework.Response, error) {
	r.mu.Lock()
	r.chores = append(r.chores, chores.Chores...)
	r.mu.Unlock()

	return &housework.Response{Message: "ok"}, nil
}

func (r *Rosie) Complete(_ context.Context,
	req *housework.CompleteRequest) (*housework.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.chores == nil || req.ChoreNumber < 1 ||
		int(req.ChoreNumber) > len(r.chores) {
		return nil, fmt.Errorf("chore %d not found", req.ChoreNumber)
	}

	r.chores[req.ChoreNumber-1].Complete = true

	return &housework.Response{Message: "ok"}, nil
}

func (r *Rosie) List(_ context.Context, _ *housework.Empty) (
	*housework.Chores, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.chores == nil {
		r.chores = make([]*housework.Chore, 0)
	}

	return &housework.Chores{Chores: r.chores}, nil
}

// Rosie의 Add, Complete, List 메서드를 포함하는 새로운 housework.RobotMaidService 인스턴스의 포인터를 반환
func (r *Rosie) Service() *housework.RobotMaidService {
	return &housework.RobotMaidService{
		Add:      r.Add,
		Complete: r.Complete,
		List:     r.List,
	}
}
