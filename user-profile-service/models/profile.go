package models

import (
	"strconv"
	"time"
)

type ExperienceLevel string

const (
	ExperienceEntry  ExperienceLevel = "ENTRY"
	ExperienceJunior ExperienceLevel = "JUNIOR"
	ExperienceMid    ExperienceLevel = "MID"
	ExperienceSenior ExperienceLevel = "SENIOR"
	ExperienceExpert ExperienceLevel = "EXPERT"
)

type ReviewType string

const (
	ReviewEmployerToHunter ReviewType = "EMPLOYER_TO_HUNTER"
	ReviewHunterToEmployer ReviewType = "HUNTER_TO_EMPLOYER"
)

type CreateProfileParams struct {
	ExpectedSalaryMin string          `json:"expected_salary_min"`
	ExpectedSalaryMax string         `json:"expected_salary_max"`
	WorkLocation     string          `json:"work_location"`
	ExperienceLevel  ExperienceLevel `json:"experience_level"`
	Bio              string          `json:"bio"`
	AvatarURL        string          `json:"avatar_url"`
}

type UpdateProfileParams struct {
	ExpectedSalaryMin string          `json:"expected_salary_min"`
	ExpectedSalaryMax string          `json:"expected_salary_max"`
	WorkLocation     string          `json:"work_location"`
	ExperienceLevel  ExperienceLevel `json:"experience_level"`
	Bio              string          `json:"bio"`
	AvatarURL        string          `json:"avatar_url"`
	Role             string          `json:"role"`
}

type RefreshStatsParams struct {
	BountyID                  int64 `json:"bounty_id"`
	DeltaCompleted            int32 `json:"delta_completed"`
	DeltaEarnings            int64 `json:"delta_earnings"`
	DeltaPosted              int32 `json:"delta_posted"`
	DeltaCompletedAsEmployer int32 `json:"delta_completed_as_employer"`
}

// UserProfile GORM model
type UserProfile struct {
	Username                string          `gorm:"type:varchar(255);primaryKey"`
	ExpectedSalaryMin      string          `gorm:"type:varchar(50);default:'0'"`
	ExpectedSalaryMax      string          `gorm:"type:varchar(50);default:'0'"`
	WorkLocation           string          `gorm:"type:varchar(255);default:''"`
	ExperienceLevel        ExperienceLevel `gorm:"type:varchar(20);default:'ENTRY'"`
	Bio                    string          `gorm:"type:text;default:''"`
	AvatarURL              string          `gorm:"type:varchar(500);default:''"`
	CompletionRate         float64         `gorm:"type:double precision;default:0.0"`
	GoodReviewRate         float64         `gorm:"type:double precision;default:0.0"`
	TotalBountiesPosted    int             `gorm:"default:0"`
	TotalBountiesCompleted int             `gorm:"default:0"`
	TotalBountiesCompletedAsEmployer int             `gorm:"default:0"`
	TotalEarnings          int64           `gorm:"default:0"`
	Role                   string          `gorm:"type:varchar(20);default:'POSTER'"`
	ReputationScore        float64         `gorm:"type:double precision;default:100.0"`
	LastCompletedAt        *time.Time      `gorm:"type:timestamp"`

	// 猎人信誉分（独立计算，独立衰减）
	HunterReputationScore float64 `gorm:"type:double precision;default:100.0"`
	// 商家信誉分（独立计算，独立衰减）
	EmployerReputationScore float64 `gorm:"type:double precision;default:100.0"`
	// 最后活跃时间（用于冷却计算）
	LastActiveAt *time.Time `gorm:"type:timestamp"`
	// 猎人累计好评数
	TotalGoodReviews int `gorm:"default:0"`
	// 猎人累计差评数
	TotalBadReviews int `gorm:"default:0"`
	// 猎人平均评分（所有评价的平均）
	AverageRating float64 `gorm:"type:double precision;default:0.0"`
	// 冷却衰减率参数（λ），默认 0.05（5%/天）
	CoolingLambda float64 `gorm:"type:double precision;default:0.05"`

	// 履约指数 [0,100]，默认 50
	HunterFulfillmentIndex   int `gorm:"default:50"`
	EmployerFulfillmentIndex int `gorm:"default:50"`
	// 履约指数窗口大小
	TaskWindowSize int `gorm:"default:50"`
	// 乐观锁版本号
	Version int `gorm:"default:1"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (p *UserProfile) GetExpectedSalaryMinInt() int64 {
	v, _ := strconv.ParseInt(p.ExpectedSalaryMin, 10, 64)
	return v
}

func (p *UserProfile) GetExpectedSalaryMaxInt() int64 {
	v, _ := strconv.ParseInt(p.ExpectedSalaryMax, 10, 64)
	return v
}
