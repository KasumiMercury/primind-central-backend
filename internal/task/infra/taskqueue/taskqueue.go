package taskqueue

import "context"

//go:generate mockgen -source=taskqueue.go -destination=mock_taskqueue.go -package=taskqueue

type RemindQueue interface {
	RegisterRemind(ctx context.Context, req *CreateRemindRequest) (*RemindResponse, error)
}
