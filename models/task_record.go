package models

import "time"

// TaskRecordRole represents the role a user takes in a task record
type TaskRecordRole string

const (
	RoleHunter   TaskRecordRole = "HUNTER"
	RoleEmployer TaskRecordRole = "EMPLOYER"
)

// TaskRecordOutcome represents the outcome of a task
type TaskRecordOutcome int

const (
	OutcomeCompleted     TaskRecordOutcome = 1
	OutcomeNeutral      TaskRecordOutcome = 0  // REJECTED or CANCELED
	OutcomeExpired      TaskRecordOutcome = -1 // DEADLINE_MISSED
)

// TaskRecord 存储每次任务流转至终态时的履约计算记录
type TaskRecord struct {
	ID               int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Username         string           `gorm:"type:varchar(255);not null;index" json:"username"`
	Role             TaskRecordRole   `gorm:"type:varchar(20);not null;index" json:"role"`
	BountyID         int64            `gorm:"not null;index" json:"bounty_id"`
	Amount           int64            `gorm:"not null" json:"amount"` // 任务标的金额（cent）
	Outcome          TaskRecordOutcome `gorm:"not null" json:"outcome"`
	OutcomeDetail    string           `gorm:"type:varchar(100)" json:"outcome_detail,omitempty"`
	EmployerRating   int              `gorm:"default:3" json:"employer_rating"`
	HunterRating     int              `gorm:"default:3" json:"hunter_rating"`
	DeadlineBefore   *time.Time       `gorm:"type:timestamp" json:"deadline_before,omitempty"`
	DeadlineAfter    *time.Time       `gorm:"type:timestamp" json:"deadline_after,omitempty"`
	ExtendCount      int              `gorm:"default:0" json:"extend_count"`
	RatingFinalized  bool             `gorm:"default:false" json:"rating_finalized"`
	CreatedAt        time.Time        `gorm:"autoCreateTime" json:"created_at"`
}

func (TaskRecord) TableName() string {
	return "task_records"
}
