package worker

import (
	"context"
	"log"
	"time"

	"github.com/grayfalcon666/user-profile-service/db"
)

const ReviewSettlementDays = 7

// ReviewSettlerWorker 扫描已 COMPLETED 但评分未结算的悬赏，7 天后自动赋默认值 3 星
type ReviewSettlerWorker struct {
	store        db.Store
	pollInterval time.Duration
	stopCh       chan struct{}
}

func NewReviewSettlerWorker(store db.Store, pollInterval time.Duration) *ReviewSettlerWorker {
	return &ReviewSettlerWorker{
		store:        store,
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
	}
}

func (w *ReviewSettlerWorker) Start(ctx context.Context) {
	log.Printf("[ReviewSettler Worker] 已启动，轮询间隔: %v\n", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("收到上下文取消信号，ReviewSettler Worker 正在退出...")
			w.stopCh <- struct{}{}
			return
		case <-ticker.C:
			w.processPendingReviews(ctx)
		}
	}
}

func (w *ReviewSettlerWorker) processPendingReviews(ctx context.Context) {
	deadline := time.Now().AddDate(0, 0, -ReviewSettlementDays)

	records, err := w.store.GetUnsettledTaskRecordsSince(ctx, deadline)
	if err != nil {
		log.Printf("查询未结算评价失败: %v\n", err)
		return
	}

	if len(records) == 0 {
		return
	}

	log.Printf("发现 %d 个待结算评价记录\n", len(records))

	for _, record := range records {
		if record.RatingFinalized {
			continue
		}

		// 超时未评价，赋默认值 3 星（已由 task_record 默认值处理，此处仅标记为已结算）
		if err := w.store.SettleTaskRecordRating(ctx, record.ID, record.EmployerRating, record.HunterRating, true); err != nil {
			log.Printf("结算评价失败 [bounty_id=%d, username=%s]: %v\n",
				record.BountyID, record.Username, err)
			continue
		}

		// 触发二次履约指数重算
		score, err := w.store.RecalculateFulfillmentIndex(ctx, record.Username, string(record.Role))
		if err != nil {
			log.Printf("二次履约重算失败 [username=%s, role=%s]: %v\n",
				record.Username, record.Role, err)
		} else {
			log.Printf("7天超时结算成功，履约指数已更新: username=%s, role=%s, score=%d\n",
				record.Username, record.Role, score)
		}
	}
}

func (w *ReviewSettlerWorker) Stop() {
	close(w.stopCh)
}
