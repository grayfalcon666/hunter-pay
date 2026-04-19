package db

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/grayfalcon666/user-profile-service/domain"
	"github.com/grayfalcon666/user-profile-service/models"
	"gorm.io/gorm"
)

type Store interface {
	CreateProfile(ctx context.Context, username string, req *models.CreateProfileParams) (*models.UserProfile, error)
	GetProfile(ctx context.Context, username string) (*models.UserProfile, error)
	UpdateProfile(ctx context.Context, username string, req *models.UpdateProfileParams) (*models.UserProfile, error)
	UpdateRole(ctx context.Context, username string, role string) (*models.UserProfile, error)
	RefreshStats(ctx context.Context, username string, req *models.RefreshStatsParams) (*models.UserProfile, error)

	CreateReview(ctx context.Context, reviewerUsername string, req *models.CreateReviewParams) (*models.UserReview, error)
	GetReviewsByUser(ctx context.Context, username string) ([]models.UserReview, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]models.UserProfile, error)
	SearchHunters(ctx context.Context, query string, limit int, sortBy string, excludeUsername string) ([]models.UserProfile, error)

	// 用户初始化相关方法
	GetProfileByInitRequestID(ctx context.Context, requestId string) (string, error)
	UpdateProfileInitRequestID(ctx context.Context, username, requestId string) error

	// 幂等性相关方法
	IsProfileEventProcessed(ctx context.Context, requestId string) (bool, error)
	RecordProcessedProfileEvent(ctx context.Context, requestId, username string, bountyId int64) error

	// 履约指数相关方法
	GetTaskRecords(ctx context.Context, username string, role string, limit int) ([]models.TaskRecord, error)
	RecalculateFulfillmentIndex(ctx context.Context, username string, role string) (int, error)
	UpdateTaskRecordRating(ctx context.Context, bountyID int64, taskUsername string, reviewerRole models.ReviewType, rating int) error
	GetTaskRecordByBountyAndUsername(ctx context.Context, bountyID int64, username string) (*models.TaskRecord, error)
	SettleTaskRecordRating(ctx context.Context, recordID int64, employerRating, hunterRating int, finalized bool) error

	// Worker support methods
	GetProfilesInactiveSince(ctx context.Context, since time.Time) ([]models.UserProfile, error)
	UpdateFulfillmentIndexWithVersion(ctx context.Context, username string, newHunterScore, newEmployerScore int, version int) error
	GetUnsettledTaskRecordsSince(ctx context.Context, since time.Time) ([]models.TaskRecord, error)
}

type SQLStore struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &SQLStore{db: db}
}

func (store *SQLStore) CreateProfile(ctx context.Context, username string, req *models.CreateProfileParams) (*models.UserProfile, error) {
	profile := &models.UserProfile{
		Username:            username,
		ExpectedSalaryMin:   req.ExpectedSalaryMin,
		ExpectedSalaryMax:   req.ExpectedSalaryMax,
		WorkLocation:        req.WorkLocation,
		ExperienceLevel:     req.ExperienceLevel,
		Bio:                 req.Bio,
		AvatarURL:           req.AvatarURL,
	}
	err := store.db.WithContext(ctx).Create(profile).Error
	return profile, err
}

func (store *SQLStore) GetProfile(ctx context.Context, username string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := store.db.WithContext(ctx).Where("username = ?", username).First(&profile).Error
	if err != nil {
		return nil, err
	}
	// Apply Newton's cooling law decay to both hunter and employer reputation scores
	profile.HunterReputationScore = CalculateDecayedReputationScore(
		profile.TotalBountiesCompleted,
		profile.TotalGoodReviews,
		profile.TotalBadReviews,
		profile.AverageRating,
		profile.TotalEarnings,
		profile.LastActiveAt,
		profile.CoolingLambda,
	)
	profile.EmployerReputationScore = CalculateDecayedReputationScore(
		profile.TotalBountiesCompletedAsEmployer,
		profile.TotalGoodReviews,
		profile.TotalBadReviews,
		profile.AverageRating,
		0,
		profile.LastActiveAt,
		profile.CoolingLambda,
	)
	return &profile, nil
}

// CalculateDecayedReputationScore applies Newton's cooling law decay to a user's reputation.
//
// Formula:
//   base_score = good_reviews×10 - bad_reviews×15 + avg_rating×completed×2 + earnings×0.001
//   decayed_score = base_score × exp(-λ × inactive_days)
//
// Parameters:
//   - completed: total bounties completed
//   - goodReviews: reviews with rating >= 4
//   - badReviews: reviews with rating <= 2
//   - avgRating: average rating (1-5)
//   - earnings: total earnings in cents
//   - lastActive: last active timestamp
//   - lambda: cooling decay rate (default 0.05 = 5% per day)
func CalculateDecayedReputationScore(completed int, goodReviews, badReviews int, avgRating float64, earnings int64, lastActive *time.Time, lambda float64) float64 {
	if lambda <= 0 {
		lambda = 0.05
	}
	if completed == 0 && goodReviews == 0 {
		return 100.0 // Default starting reputation for new users
	}

	// Base score components
	baseScore := float64(goodReviews)*10.0 -
		float64(badReviews)*15.0 +
		avgRating*float64(completed)*2.0 +
		float64(earnings)*0.001

	// Apply minimum floor
	if baseScore < 0 {
		baseScore = 0
	}

	// Newton's cooling decay
	if lastActive == nil {
		return baseScore
	}
	inactiveDays := time.Since(*lastActive).Hours() / 24.0
	if inactiveDays < 0 {
		inactiveDays = 0
	}
	decayed := baseScore * math.Exp(-lambda*inactiveDays)
	if decayed < 0 {
		decayed = 0
	}
	return decayed
}

func (store *SQLStore) UpdateProfile(ctx context.Context, username string, req *models.UpdateProfileParams) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := store.db.WithContext(ctx).Where("username = ?", username).First(&profile).Error
	if err != nil {
		return nil, err
	}

	if req.ExpectedSalaryMin != "" {
		profile.ExpectedSalaryMin = req.ExpectedSalaryMin
	}
	if req.ExpectedSalaryMax != "" {
		profile.ExpectedSalaryMax = req.ExpectedSalaryMax
	}
	if req.WorkLocation != "" {
		profile.WorkLocation = req.WorkLocation
	}
	if req.ExperienceLevel != "" {
		profile.ExperienceLevel = req.ExperienceLevel
	}
	if req.Bio != "" {
		profile.Bio = req.Bio
	}
	if req.AvatarURL != "" {
		profile.AvatarURL = req.AvatarURL
	}

	err = store.db.WithContext(ctx).Save(&profile).Error
	return &profile, err
}

func (store *SQLStore) RefreshStats(ctx context.Context, username string, req *models.RefreshStatsParams) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := store.db.WithContext(ctx).Where("username = ?", username).First(&profile).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if req.DeltaCompleted > 0 {
		profile.TotalBountiesCompleted += int(req.DeltaCompleted)
		profile.TotalEarnings += req.DeltaEarnings
		if profile.TotalBountiesPosted+profile.TotalBountiesCompleted > 0 {
			profile.CompletionRate = float64(profile.TotalBountiesCompleted) / float64(profile.TotalBountiesPosted+profile.TotalBountiesCompleted)
		}
		profile.LastCompletedAt = &now
		profile.LastActiveAt = &now
	} else if req.DeltaCompleted < 0 {
		if profile.TotalBountiesCompleted > 0 {
			profile.TotalBountiesCompleted--
			if profile.TotalBountiesPosted+profile.TotalBountiesCompleted > 0 {
				profile.CompletionRate = float64(profile.TotalBountiesCompleted) / float64(profile.TotalBountiesPosted+profile.TotalBountiesCompleted)
			}
		}
	}

	if req.DeltaPosted > 0 {
		profile.TotalBountiesPosted += int(req.DeltaPosted)
		if profile.TotalBountiesPosted+profile.TotalBountiesCompleted > 0 {
			profile.CompletionRate = float64(profile.TotalBountiesCompleted) / float64(profile.TotalBountiesPosted+profile.TotalBountiesCompleted)
		}
	}

	if req.DeltaCompletedAsEmployer > 0 {
		profile.TotalBountiesCompletedAsEmployer += int(req.DeltaCompletedAsEmployer)
		profile.LastActiveAt = &now
	}

	err = store.db.WithContext(ctx).Save(&profile).Error
	return &profile, err
}

func (store *SQLStore) CreateReview(ctx context.Context, reviewerUsername string, req *models.CreateReviewParams) (*models.UserReview, error) {
	review := &models.UserReview{
		ReviewerUsername:  reviewerUsername,
		ReviewedUsername: req.ReviewedUsername,
		BountyID:         req.BountyID,
		Rating:           req.Rating,
		Comment:          req.Comment,
		ReviewType:       req.ReviewType,
	}
	err := store.db.WithContext(ctx).Create(review).Error
	if err != nil {
		return nil, err
	}

	// 更新被评价人的统计
	var allReviews []models.UserReview
	store.db.WithContext(ctx).Where("reviewed_username = ?", req.ReviewedUsername).Find(&allReviews)
	total := len(allReviews)
	if total > 0 {
		var profile models.UserProfile
		store.db.WithContext(ctx).Where("username = ?", req.ReviewedUsername).First(&profile)

		good := 0
		bad := 0
		sum := 0
		for _, r := range allReviews {
			sum += int(r.Rating)
			if r.Rating >= 4 {
				good++
			} else if r.Rating <= 2 {
				bad++
			}
		}
		profile.GoodReviewRate = float64(good) / float64(total)
		profile.TotalGoodReviews = good
		profile.TotalBadReviews = bad
		profile.AverageRating = float64(sum) / float64(total)
		profile.LastActiveAt = &review.CreatedAt

		// 更新猎人信誉分
		profile.HunterReputationScore = CalculateDecayedReputationScore(
			profile.TotalBountiesCompleted,
			profile.TotalGoodReviews,
			profile.TotalBadReviews,
			profile.AverageRating,
			profile.TotalEarnings,
			profile.LastActiveAt,
			profile.CoolingLambda,
		)
		// 更新商家信誉分
		profile.EmployerReputationScore = CalculateDecayedReputationScore(
			profile.TotalBountiesCompletedAsEmployer,
			profile.TotalGoodReviews,
			profile.TotalBadReviews,
			profile.AverageRating,
			0,
			profile.LastActiveAt,
			profile.CoolingLambda,
		)

		store.db.WithContext(ctx).Save(&profile)
	}

	return review, err
}

func (store *SQLStore) GetReviewsByUser(ctx context.Context, username string) ([]models.UserReview, error) {
	var reviews []models.UserReview
	err := store.db.WithContext(ctx).Where("reviewed_username = ?", username).Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (store *SQLStore) SearchUsers(ctx context.Context, query string, limit int) ([]models.UserProfile, error) {
	var profiles []models.UserProfile
	if limit <= 0 {
		limit = 10
	}
	err := store.db.WithContext(ctx).
		Where("username ILIKE ?", "%"+query+"%").
		Limit(limit).
		Find(&profiles).Error
	return profiles, err
}

// GetProfileByInitRequestID 根据初始化请求 ID 查找用户
func (store *SQLStore) GetProfileByInitRequestID(ctx context.Context, requestId string) (string, error) {
	var username string
	err := store.db.WithContext(ctx).
		Table("user_profiles").
		Where("initialization_request_id = ?", requestId).
		Pluck("username", &username).
		Error
	return username, err
}

// UpdateProfileInitRequestID 更新用户资料的初始化请求 ID
func (store *SQLStore) UpdateProfileInitRequestID(ctx context.Context, username, requestId string) error {
	return store.db.WithContext(ctx).
		Table("user_profiles").
		Where("username = ?", username).
		Update("initialization_request_id", requestId).
		Error
}

func (store *SQLStore) UpdateRole(ctx context.Context, username string, role string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := store.db.WithContext(ctx).Where("username = ?", username).First(&profile).Error
	if err != nil {
		return nil, err
	}

	profile.Role = role
	err = store.db.WithContext(ctx).Save(&profile).Error
	return &profile, err
}

func (store *SQLStore) SearchHunters(ctx context.Context, query string, limit int, sortBy string, excludeUsername string) ([]models.UserProfile, error) {
	var profiles []models.UserProfile
	if limit <= 0 {
		limit = 20
	}

	db := store.db.WithContext(ctx).
		Where("username ILIKE ?", "%"+query+"%")
	if excludeUsername != "" {
		db = db.Where("username != ?", excludeUsername)
	}
	db = db.Limit(limit)

	switch sortBy {
	case "good_review_rate":
		db = db.Order("good_review_rate DESC")
	case "total_bounties_completed":
		db = db.Order("total_bounties_completed DESC")
	case "reputation":
		db = db.Order("hunter_reputation_score DESC")
	case "fulfillment":
		db = db.Order("fulfillment_index DESC")
	case "completion_rate":
	default:
		db = db.Order("completion_rate DESC")
	}

	err := db.Find(&profiles).Error
	return profiles, err
}

// ==========================================
// Task Records & Fulfillment Index (履约指数)
// ==========================================

func (store *SQLStore) GetTaskRecords(ctx context.Context, username string, role string, limit int) ([]models.TaskRecord, error) {
	var records []models.TaskRecord
	if limit <= 0 {
		limit = 50
	}
	db := store.db.WithContext(ctx).
		Where("username = ? AND role = ?", username, role).
		Order("created_at DESC").
		Limit(limit)
	err := db.Find(&records).Error
	return records, err
}

func (store *SQLStore) RecalculateFulfillmentIndex(ctx context.Context, username string, role string) (int, error) {
	// 使用乐观锁更新，避免并发丢失更新
	var profile models.UserProfile
	if err := store.db.WithContext(ctx).Where("username = ?", username).First(&profile).Error; err != nil {
		return 50, err
	}

	// 获取该用户该角色的任务记录
	records, err := store.GetTaskRecords(ctx, username, role, profile.TaskWindowSize)
	if err != nil {
		return 50, err
	}

	// 转换为计算器所需的格式
	var calcRecords []domain.TaskRecordForCalc
	for _, r := range records {
		// 根据角色取对端评分：HUNTER 记录取雇主给的 EmployerRating，EMPLOYER 记录取猎人给的 HunterRating
		counterpartRating := r.EmployerRating
		if r.Role == models.RoleEmployer {
			counterpartRating = r.HunterRating
		}
		calcRecords = append(calcRecords, domain.TaskRecordForCalc{
			Amount:            r.Amount,
			Outcome:           int(r.Outcome),
			Role:              string(r.Role),
			CounterpartRating: counterpartRating,
			ExtendCount:       r.ExtendCount,
		})
	}

	calc := domain.NewFulfillmentScoreCalculator()
	newScore := calc.Calculate(calcRecords)

	// 乐观锁更新，根据 role 选择对应列
	column := "hunter_fulfillment_index"
	if role == string(models.RoleEmployer) {
		column = "employer_fulfillment_index"
	}
	result := store.db.WithContext(ctx).
		Model(&profile).
		Where("version = ?", profile.Version).
		Updates(map[string]interface{}{
			column:         newScore,
			"last_active_at": time.Now(),
			"version":       profile.Version + 1,
		})
	if result.Error != nil {
		return 50, result.Error
	}
	if result.RowsAffected == 0 {
		// 并发冲突，重试一次
		return store.RecalculateFulfillmentIndex(ctx, username, role)
	}

	return newScore, nil
}

func (store *SQLStore) GetTaskRecordByBountyAndUsername(ctx context.Context, bountyID int64, username string) (*models.TaskRecord, error) {
	var record models.TaskRecord
	err := store.db.WithContext(ctx).
		Where("bounty_id = ? AND username = ?", bountyID, username).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateTaskRecordRating 更新 task_record 中的评分字段
// reviewerRole 表示评价人的角色：EMPLOYER_TO_HUNTER 或 HUNTER_TO_EMPLOYER
// 根据 reviewerRole 决定更新 hunter_rating 还是 employer_rating
// 只有当对端评分也已存在时，才设置 rating_finalized=true；同时更新评价人和被评价人的记录
func (store *SQLStore) UpdateTaskRecordRating(ctx context.Context, bountyID int64, taskUsername string, reviewerRole models.ReviewType, rating int) error {
	// 找到被评价人的 task record
	var record models.TaskRecord
	if err := store.db.WithContext(ctx).
		Where("bounty_id = ? AND username = ?", bountyID, taskUsername).
		First(&record).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{}

	// 根据评价人的角色，决定更新哪个评分字段
	if reviewerRole == models.ReviewEmployerToHunter {
		// 雇主评价猎人 → 更新 hunter task_record 的 employer_rating
		updates["employer_rating"] = rating
	} else {
		// 猎人评价雇主 → 更新 employer task_record 的 hunter_rating
		updates["hunter_rating"] = rating
	}

	// 查询对端（评价人）的 task record
	counterpartRole := models.RoleHunter
	if record.Role == models.RoleHunter {
		counterpartRole = models.RoleEmployer
	}
	var counterpartRecord models.TaskRecord
	if err := store.db.WithContext(ctx).
		Where("bounty_id = ? AND role = ?", bountyID, counterpartRole).
		First(&counterpartRecord).Error; err != nil {
		return err
	}

	// 检查对端是否已有评分（!= 默认3星）
	counterpartHasRating := false
	if counterpartRecord.Role == models.RoleHunter && counterpartRecord.EmployerRating != 3 {
		counterpartHasRating = true
	} else if counterpartRecord.Role == models.RoleEmployer && counterpartRecord.HunterRating != 3 {
		counterpartHasRating = true
	}

	// 双方都评分了 → 两条记录的 rating_finalized 都设为 true
	if counterpartHasRating {
		updates["rating_finalized"] = true
	}

	// 更新被评价人的记录
	if err := store.db.WithContext(ctx).
		Model(&record).
		Updates(updates).Error; err != nil {
		return err
	}

	// 如果双方都评分了，同时把评价人的记录也标记 finalized=true
	if counterpartHasRating {
		return store.db.WithContext(ctx).
			Model(&counterpartRecord).
			Update("rating_finalized", true).Error
	}

	return nil
}


// IsProfileEventProcessed 检查是否已处理过该事件
func (store *SQLStore) IsProfileEventProcessed(ctx context.Context, requestId string) (bool, error) {
	var count int64
	err := store.db.WithContext(ctx).
		Model(&models.ProcessedProfileEvent{}).
		Where("request_id = ?", requestId).
		Count(&count).Error
	return count > 0, err
}

// RecordProcessedProfileEvent 记录已处理的事件
func (store *SQLStore) RecordProcessedProfileEvent(ctx context.Context, requestId, username string, bountyId int64) error {
	event := &models.ProcessedProfileEvent{
		RequestID: requestId,
		Username:  username,
		BountyID:  bountyId,
	}
	return store.db.WithContext(ctx).Create(event).Error
}

// ==========================================
// Worker 支持方法
// ==========================================

func (store *SQLStore) GetProfilesInactiveSince(ctx context.Context, since time.Time) ([]models.UserProfile, error) {
	var profiles []models.UserProfile
	err := store.db.WithContext(ctx).
		Where("last_active_at IS NOT NULL AND last_active_at < ?", since).
		Where("hunter_fulfillment_index != 50 OR employer_fulfillment_index != 50").
		Find(&profiles).Error
	return profiles, err
}

// UpdateFulfillmentIndexWithVersion updates both fulfillment indices with optimistic locking.
// Pass -1 for scores that should not be updated.
func (store *SQLStore) UpdateFulfillmentIndexWithVersion(ctx context.Context, username string, newHunterScore, newEmployerScore int, version int) error {
	updates := map[string]interface{}{
		"version": version + 1,
	}
	if newHunterScore >= 0 {
		updates["hunter_fulfillment_index"] = newHunterScore
	}
	if newEmployerScore >= 0 {
		updates["employer_fulfillment_index"] = newEmployerScore
	}
	result := store.db.WithContext(ctx).
		Model(&models.UserProfile{}).
		Where("username = ? AND version = ?", username, version).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("乐观锁冲突: username=%s", username)
	}
	return nil
}

func (store *SQLStore) GetUnsettledTaskRecordsSince(ctx context.Context, since time.Time) ([]models.TaskRecord, error) {
	var records []models.TaskRecord
	err := store.db.WithContext(ctx).
		Where("rating_finalized = FALSE AND created_at < ?", since).
		Find(&records).Error
	return records, err
}

func (store *SQLStore) SettleTaskRecordRating(ctx context.Context, recordID int64, employerRating, hunterRating int, finalized bool) error {
	return store.db.WithContext(ctx).
		Model(&models.TaskRecord{}).
		Where("id = ?", recordID).
		Updates(map[string]interface{}{
			"rating_finalized": finalized,
			"employer_rating":  employerRating,
			"hunter_rating":    hunterRating,
		}).Error
}
