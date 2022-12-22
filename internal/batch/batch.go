package batch

import (
	"context"
	"sync"
	"time"
)

type BatchProcess struct {
	BatchFunction func(*sync.Mutex)
	RunEvery      time.Duration
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (batchProess BatchProcess) Run() {
	batchProess.RunWithContext(context.Background())
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (batchProess BatchProcess) RunWithContext(ctx context.Context) {
	var m sync.Mutex
mainloop:
	for {
		select {
		case <-ctx.Done():
			break mainloop
		default:
			time.Sleep(batchProess.RunEvery)
			batchProess.BatchFunction(&m)
		}

	}
}
