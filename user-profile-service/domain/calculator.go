package domain

import "math"

// FulfillmentScoreCalculator 履约指数计算器，纯领域服务，无任何数据库 I/O 依赖
type FulfillmentScoreCalculator struct {
	Lambda      float64
	WindowSize  int
	Anchor      float64
	ScaleFactor float64
}

// TaskRecordForCalc 用于计算的任务记录（只含计算所需字段）
type TaskRecordForCalc struct {
	Amount   int64
	Outcome  int // +1, 0, -1
	Role     string // "HUNTER" | "EMPLOYER"
	// 对端评价（对于 HUNTER 记录取 EmployerRating，对于 EMPLOYER 记录取 HunterRating）
	CounterpartRating int // 1-5
	ExtendCount       int
}

// NewFulfillmentScoreCalculator 创建默认配置的计算器
func NewFulfillmentScoreCalculator() *FulfillmentScoreCalculator {
	return &FulfillmentScoreCalculator{
		Lambda:      0.1,
		WindowSize:  50,
		Anchor:      5.0,
		ScaleFactor: 28.0,
	}
}

// Calculate 根据任务记录列表计算履约指数，返回 [0,100] 整数
func (c *FulfillmentScoreCalculator) Calculate(records []TaskRecordForCalc) int {
	if len(records) == 0 {
		return 50
	}

	recent := records
	if len(recent) > c.WindowSize {
		recent = records[:c.WindowSize]
	}

	var weightedSum, weightSum float64
	for i, rec := range recent {
		// W_i = log10(Amount_元 + 10) * e^(-lambda * i)
		// Amount 在 DB 中以分为单位存储，先转换为元再求对数
		amountYuan := float64(rec.Amount) / 100.0
		weight := math.Log10(amountYuan+10.0) * math.Exp(-c.Lambda*float64(i))

		var v float64
		if rec.Outcome == -1 {
			// 严重违约：强制极小值
			v = -1.5
		} else {
			v = float64(rec.Outcome)
			if rec.CounterpartRating > 0 {
				// Rating 修正：(rating - 3) * 0.4
				v += (float64(rec.CounterpartRating) - 3.0) * 0.4
			}
			// 延期惩罚
			v -= float64(rec.ExtendCount) * 0.15
		}

		weightedSum += weight * v
		weightSum += weight
	}

	denom := math.Max(weightSum, c.Anchor)
	score := (weightedSum / denom) * c.ScaleFactor + 50.0
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}
