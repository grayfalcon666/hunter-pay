package mq

import (
	"context"
	"log"
	"time"

	"github.com/grayfalcon666/escrow-bounty/db"
)

// OutboxWorker 负责扫描 outbox 表并发布消息到 RabbitMQ
type OutboxWorker struct {
	store     db.Store
	producer  *ProfileUpdateProducer
	pollInterval time.Duration
	batchSize    int
	stopCh       chan struct{}
}

// NewOutboxWorker 创建 OutboxWorker
func NewOutboxWorker(store db.Store, producer *ProfileUpdateProducer, pollInterval time.Duration, batchSize int) *OutboxWorker {
	return &OutboxWorker{
		store:        store,
		producer:     producer,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动 worker
func (w *OutboxWorker) Start(ctx context.Context) {
	log.Printf("[Outbox Worker] 已启动，轮询间隔: %v, 批次大小: %d\n", w.pollInterval, w.batchSize)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("收到上下文取消信号，Outbox Worker 正在退出...")
			w.stopCh <- struct{}{}
			return
		case <-ticker.C:
			log.Printf("[Outbox Worker] 收到定时器信号，开始处理...\n")
			w.processOutbox(ctx)
		}
	}
}

// processOutbox 处理待发送的 outbox 记录
func (w *OutboxWorker) processOutbox(ctx context.Context) {
	log.Printf("[Outbox Worker] 开始扫描待处理记录...\n")
	entries, err := w.store.GetPendingOutboxEntries(ctx, w.batchSize)
	if err != nil {
		log.Printf("查询 outbox 记录失败: %v\n", err)
		return
	}

	log.Printf("[Outbox Worker] 查询到 %d 条待处理记录\n", len(entries))

	if len(entries) == 0 {
		return
	}

	log.Printf("发现 %d 条待处理的 outbox 记录\n", len(entries))

	for _, entry := range entries {
		event := &ProfileUpdateEvent{
			Username:                  entry.Username,
			BountyID:                  entry.BountyID,
			DeltaCompleted:            entry.DeltaCompleted,
			DeltaEarnings:             entry.DeltaEarnings,
			DeltaPosted:               entry.DeltaPosted,
			DeltaCompletedAsEmployer:   entry.DeltaCompletedAsEmployer,
			RequestID:                 entry.RequestID,
		}

		publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := w.producer.Publish(publishCtx, event)
		cancel()

		if err != nil {
			log.Printf("发布 outbox 消息失败 [id=%d, username=%s]: %v\n", entry.ID, entry.Username, err)
			// 标记为失败，等待重试
			if markErr := w.store.MarkOutboxFailed(ctx, entry.ID, err.Error()); markErr != nil {
				log.Printf("标记 outbox 失败状态失败 [id=%d]: %v\n", entry.ID, markErr)
			}
			continue
		}

		// 发布成功，标记为完成
		log.Printf("准备标记 outbox [id=%d] 为 COMPLETED\n", entry.ID)
		if err := w.store.MarkOutboxCompleted(ctx, entry.ID); err != nil {
			log.Printf("标记 outbox 完成状态失败 [id=%d]: %v\n", entry.ID, err)
		} else {
			log.Printf("Outbox 消息已成功发布 [id=%d, username=%s, bounty_id=%d] 已标记 COMPLETED\n",
				entry.ID, entry.Username, entry.BountyID)
		}
	}
}

// Stop 停止 worker
func (w *OutboxWorker) Stop() {
	close(w.stopCh)
}
