package remindregister

import "context"

//go:generate mockgen -source=queue.go -destination=mock_queue.go -package=remindregister

type Queue interface {
	RegisterRemind(ctx context.Context, req *CreateRemindRequest) (*RemindResponse, error)
}
