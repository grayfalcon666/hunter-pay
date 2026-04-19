package worker

import (
	"context"
	"log"
	"time"

	"github.com/grayfalcon666/user-profile-service/db"
)

const (
	InactiveThresholdDays = 90
	DecayPercentPerDay   = 0.005 // 0.5% 向 50 分靠拢
)

// InactivityDecayWorker 每日扫描长期不活跃用户，将履约指数向 50 分衰减
type InactivityDecayWorker struct {
	store        db.Store
	pollInterval time.Duration
	stopCh       chan struct{}
}

func NewInactivityDecayWorker(store db.Store, pollInterval time.Duration) *InactivityDecayWorker {
	return &InactivityDecayWorker{
		store:        store,
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
	}
}

func (w *InactivityDecayWorker) Start(ctx context.Context) {
	log.Printf("[InactivityDecay Worker] 已启动，轮询间隔: %v\n", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("收到上下文取消信号，InactivityDecay Worker 正在退出...")
			w.stopCh <- struct{}{}
			return
		case <-ticker.C:
			w.processInactiveProfiles(ctx)
		}
	}
}

func (w *InactivityDecayWorker) processInactiveProfiles(ctx context.Context) {
	threshold := time.Now().AddDate(0, 0, -InactiveThresholdDays)
	profiles, err := w.store.GetProfilesInactiveSince(ctx, threshold)
	if err != nil {
		log.Printf("查询不活跃用户失败: %v\n", err)
		return
	}

	if len(profiles) == 0 {
		return
	}

	log.Printf("发现 %d 个长期不活跃用户，开始静默衰减...\n", len(profiles))

	for _, profile := range profiles {
		// 猎人履约指数衰减
		var newHunterScore int
		if profile.HunterFulfillmentIndex != 50 {
			decayed := float64(profile.HunterFulfillmentIndex-50)*(1-DecayPercentPerDay) + 50.0
			newHunterScore = int(decayed + 0.5)
			if newHunterScore > 100 {
				newHunterScore = 100
			}
		} else {
			newHunterScore = -1 // 不更新
		}

		// 雇主履约指数衰减
		var newEmployerScore int
		if profile.EmployerFulfillmentIndex != 50 {
			decayed := float64(profile.EmployerFulfillmentIndex-50)*(1-DecayPercentPerDay) + 50.0
			newEmployerScore = int(decayed + 0.5)
			if newEmployerScore > 100 {
				newEmployerScore = 100
			}
		} else {
			newEmployerScore = -1 // 不更新
		}

		// 两项都是 50，跳过
		if newHunterScore == -1 && newEmployerScore == -1 {
			continue
		}

		if err := w.store.UpdateFulfillmentIndexWithVersion(ctx, profile.Username, newHunterScore, newEmployerScore, profile.Version); err != nil {
			log.Printf("更新履约指数衰减失败 [username=%s]: %v\n", profile.Username, err)
		} else {
			log.Printf("履约指数静默衰减: username=%s, hunter: %d→%d, employer: %d→%d\n",
				profile.Username, profile.HunterFulfillmentIndex, newHunterScore,
				profile.EmployerFulfillmentIndex, newEmployerScore)
		}
	}
}

func (w *InactivityDecayWorker) Stop() {
	close(w.stopCh)
}
