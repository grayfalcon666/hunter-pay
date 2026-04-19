package worker

import (
	"context"
	"log"
	"time"

	"github.com/grayfalcon666/escrow-bounty/db"
)

// DeadlineChecker 定时检测过期悬赏并标记为 EXPIRED
type DeadlineChecker struct {
	store        db.Store
	pollInterval time.Duration
	batchSize    int
	stopCh       chan struct{}
}

func NewDeadlineChecker(store db.Store, pollInterval time.Duration, batchSize int) *DeadlineChecker {
	return &DeadlineChecker{
		store:        store,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		stopCh:       make(chan struct{}),
	}
}

func (w *DeadlineChecker) Start(ctx context.Context) {
	log.Printf("[DeadlineChecker] 已启动，轮询间隔: %v, 批次大小: %d\n", w.pollInterval, w.batchSize)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("收到上下文取消信号，DeadlineChecker 正在退出...")
			w.stopCh <- struct{}{}
			return
		case <-ticker.C:
			w.checkExpiredBounties(ctx)
		}
	}
}

func (w *DeadlineChecker) checkExpiredBounties(ctx context.Context) {
	bounties, err := w.store.ListExpiredBounties(ctx, w.batchSize)
	if err != nil {
		log.Printf("查询过期悬赏失败: %v\n", err)
		return
	}

	if len(bounties) == 0 {
		return
	}

	log.Printf("发现 %d 个过期悬赏待处理\n", len(bounties))

	for _, bounty := range bounties {
		if err := w.store.ExpireBounty(ctx, bounty.ID); err != nil {
			log.Printf("处理过期悬赏失败 [bounty_id=%d]: %v\n", bounty.ID, err)
		} else {
			log.Printf("过期悬赏已标记为 EXPIRED [bounty_id=%d]\n", bounty.ID)
		}
	}
}

func (w *DeadlineChecker) Stop() {
	close(w.stopCh)
}
