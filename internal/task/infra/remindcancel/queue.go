package remindcancel

import "context"

//go:generate mockgen -source=queue.go -destination=mock_queue.go -package=remindcancel

type Queue interface {
	CancelRemind(ctx context.Context, req *CancelRemindRequest) (*CancelRemindResponse, error)
}
