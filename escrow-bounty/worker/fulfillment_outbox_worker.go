package worker

import (
	"context"
	"log"
	"time"

	"github.com/grayfalcon666/escrow-bounty/db"
	"github.com/grayfalcon666/escrow-bounty/mq"
)

// FulfillmentOutboxWorker 负责扫描 fulfillment_outbox 表并发布履约重算事件到 RabbitMQ
type FulfillmentOutboxWorker struct {
	store        db.Store
	producer     *mq.FulfillmentRecalcProducer
	pollInterval time.Duration
	batchSize    int
	stopCh       chan struct{}
}

func NewFulfillmentOutboxWorker(store db.Store, producer *mq.FulfillmentRecalcProducer, pollInterval time.Duration, batchSize int) *FulfillmentOutboxWorker {
	return &FulfillmentOutboxWorker{
		store:        store,
		producer:     producer,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		stopCh:       make(chan struct{}),
	}
}

func (w *FulfillmentOutboxWorker) Start(ctx context.Context) {
	log.Printf("[FulfillmentOutbox Worker] 已启动，轮询间隔: %v, 批次大小: %d\n", w.pollInterval, w.batchSize)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("收到上下文取消信号，FulfillmentOutbox Worker 正在退出...")
			w.stopCh <- struct{}{}
			return
		case <-ticker.C:
			w.processOutbox(ctx)
		}
	}
}

func (w *FulfillmentOutboxWorker) processOutbox(ctx context.Context) {
	entries, err := w.store.GetPendingFulfillmentOutbox(ctx, w.batchSize)
	if err != nil {
		log.Printf("查询 fulfillment_outbox 记录失败: %v\n", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	log.Printf("发现 %d 条待处理的 fulfillment_outbox 记录\n", len(entries))

	for _, entry := range entries {
		event := &mq.FulfillmentRecalcEvent{
			Username:  entry.Username,
			Role:      entry.Role,
			BountyID:  entry.BountyID,
			RequestID: entry.RequestID,
		}

		publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := w.producer.Publish(publishCtx, event)
		cancel()

		if err != nil {
			log.Printf("发布履约重算消息失败 [id=%d, username=%s]: %v\n", entry.ID, entry.Username, err)
			if markErr := w.store.MarkFulfillmentOutboxFailed(ctx, entry.ID, err.Error()); markErr != nil {
				log.Printf("标记 fulfillment_outbox 失败状态失败 [id=%d]: %v\n", entry.ID, markErr)
			}
			continue
		}

		if err := w.store.MarkFulfillmentOutboxCompleted(ctx, entry.ID); err != nil {
			log.Printf("标记 fulfillment_outbox 完成状态失败 [id=%d]: %v\n", entry.ID, err)
		} else {
			log.Printf("履约重算消息已成功发布 [id=%d, username=%s, role=%s]\n",
				entry.ID, entry.Username, entry.Role)
		}
	}
}

func (w *FulfillmentOutboxWorker) Stop() {
	close(w.stopCh)
}
