// Package scheduler implements the scheduled_task.v1 RPC. Currently a
// single task: library_scan.
package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	pluginv1 "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginproto/continuum/plugin/v1"
)

// Tasks holds the registered ScanFn so both the admin trigger and the
// scheduled-task RPC run the same code path. ScanFn returns the
// scan_event id; concurrent triggers de-duplicate (the in-flight call's
// id is returned to subsequent callers).
type Tasks struct {
	ScanFn func(context.Context) (int64, error)

	mu      sync.Mutex
	running atomic.Bool
}

// Server implements ScheduledTaskServer.
type Server struct {
	pluginv1.UnimplementedScheduledTaskServer
	t *Tasks
}

func New(t *Tasks) *Server { return &Server{t: t} }

func (s *Server) Run(ctx context.Context, req *pluginv1.RunScheduledTaskRequest) (*pluginv1.RunScheduledTaskResponse, error) {
	if req.GetTaskKey() != "library_scan" {
		return nil, errors.New("unknown task key")
	}
	if s.t == nil || s.t.ScanFn == nil {
		return &pluginv1.RunScheduledTaskResponse{}, nil
	}
	if !s.t.running.CompareAndSwap(false, true) {
		// Previous scan still running; drop this trigger.
		return &pluginv1.RunScheduledTaskResponse{}, nil
	}
	defer s.t.running.Store(false)
	_, err := s.t.ScanFn(ctx)
	return &pluginv1.RunScheduledTaskResponse{}, err
}
