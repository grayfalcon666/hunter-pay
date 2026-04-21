package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/grayfalcon666/escrow-bounty/models"
	simplebankpb "github.com/grayfalcon666/escrow-bounty/simplebankpb"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Store interface {
	// Profile update outbox operations
	PublishProfileUpdate(ctx context.Context, outbox *models.ProfileUpdateOutbox) error
	GetPendingOutboxEntries(ctx context.Context, limit int) ([]models.ProfileUpdateOutbox, error)
	MarkOutboxCompleted(ctx context.Context, id int64) error
	MarkOutboxFailed(ctx context.Context, id int64, errMsg string) error

	CreateBounty(ctx context.Context, bounty *models.Bounty) error
	GetBountyByID(ctx context.Context, id int64) (*models.Bounty, error)
	ListBounties(ctx context.Context, status models.BountyStatus, limit, offset int) ([]models.Bounty, error)
	UpdateBountyStatus(ctx context.Context, id int64, status models.BountyStatus) error
	CreateApplication(ctx context.Context, app *models.BountyApplication) error
	UpdateApplicationStatus(ctx context.Context, applicationID int64, status models.ApplicationStatus) error
	PublishBounty(ctx context.Context, bounty *models.Bounty, bankClient BankClient, employerAccountID int64) error
	AcceptBounty(ctx context.Context, bountyID, hunter_account_id int64, hunterUsername string) (*models.BountyApplication, error)
	ConfirmHunter(ctx context.Context, bountyID int64, applicationID int64, employerUsername string) error
	SubmitBounty(ctx context.Context, bountyID int64, hunterUsername, submissionText string) error
	ApproveBounty(ctx context.Context, bountyID int64, employerUsername string, bankClient BankClient) error
	RejectBounty(ctx context.Context, bountyID int64, employerUsername string) error
	ReSubmitBounty(ctx context.Context, bountyID int64, hunterUsername, submissionText string) error
	CompleteBounty(ctx context.Context, bountyID int64, employerUsername string, bankClient BankClient) error
	CancelBounty(ctx context.Context, bountyID int64, employerUsername string, bankClient BankClient) error
	DeleteBounty(ctx context.Context, bountyID int64, employerUsername string) error
	// Chat operations
	GetOrCreateChat(ctx context.Context, bountyID int64) (*models.Chat, error)
	CreateMessage(ctx context.Context, chatID int64, senderUsername, content string) (*models.ChatMessage, error)
	ListMessages(ctx context.Context, bountyID int64) ([]models.ChatMessage, error)
	// Private chat operations
	GetOrCreatePrivateConversation(ctx context.Context, user1, user2 string) (*models.PrivateConversation, error)
	ListPrivateConversations(ctx context.Context, username string) ([]models.PrivateConversation, error)
	DeletePrivateConversation(ctx context.Context, convID int64, username string) error
	// Unified message operations
	CreateMessageV2(ctx context.Context, msgType string, convID, bountyChatID *int64, senderUsername, content string) (*models.AllMessage, error)
	ListMessagesV2(ctx context.Context, convID int64, limit, offset int) ([]models.AllMessage, error)
	ListBountyMessagesV2(ctx context.Context, bountyChatID int64, limit, offset int) ([]models.AllMessage, error)
	MarkMessagesRead(ctx context.Context, convID int64, readerUsername string) error
	GetUnreadCounts(ctx context.Context, username string) (map[int64]int, error)
	// Comment operations
	ListComments(ctx context.Context, bountyID int64) ([]models.Comment, error)
	GetComment(ctx context.Context, id int64) (*models.Comment, error)
	CreateComment(ctx context.Context, bountyID int64, replyToID *int64, authorUsername, content string, imageID *int64) (*models.Comment, error)
	DeleteComment(ctx context.Context, commentID int64, username string) error
	DeleteCommentCascade(ctx context.Context, commentID int64, username string) ([]string, error) // 返回被删除的图片相对路径
	ListCommentsByUsername(ctx context.Context, username string, limit, offset int) ([]models.Comment, error)
	// Image operations
	CreateImage(ctx context.Context, img *models.Image) error
	GetImage(ctx context.Context, id int64) (*models.Image, error)
	GetAvatarUrlsByUsernames(ctx context.Context, usernames []string) (map[string]string, error)
	DeleteImage(ctx context.Context, id int64) error
	ListImagesByEntity(ctx context.Context, entityType string, entityID string) ([]models.Image, error)
	DeleteCommentImages(ctx context.Context, commentID int64) ([]models.Image, error) // 返回被删除的图片记录（含路径）
	// Invitation operations
	CreateInvitation(ctx context.Context, inv *models.Invitation) (*models.Invitation, error)
	GetInvitationByID(ctx context.Context, id int64) (*models.Invitation, error)
	ListInvitationsByHunter(ctx context.Context, hunterUsername string, status models.InvitationStatus, limit, offset int) ([]models.Invitation, error)
	ListInvitationsByPoster(ctx context.Context, posterUsername string, bountyID *int64, status models.InvitationStatus, limit, offset int) ([]models.Invitation, error)
	UpdateInvitationStatus(ctx context.Context, id int64, status models.InvitationStatus) (*models.Invitation, error)
	DeleteInvitation(ctx context.Context, id int64, posterUsername string) error
	// Application operations
	ListApplicationsByHunter(ctx context.Context, hunterUsername string, status models.ApplicationStatus, limit, offset int) ([]models.BountyApplication, error)

	// Task record operations (履约指数)
	WriteTaskRecords(ctx context.Context, bounty *models.Bounty, hunterUsername string, outcome models.TaskRecordOutcome, outcomeDetail string) error
	ExpireBounty(ctx context.Context, bountyID int64) error
	ListExpiredBounties(ctx context.Context, limit int) ([]models.Bounty, error)

	// Fulfillment outbox operations
	PublishFulfillmentRecalc(ctx context.Context, username, role string, bountyID int64) error
	GetPendingFulfillmentOutbox(ctx context.Context, limit int) ([]models.FulfillmentOutbox, error)
	MarkFulfillmentOutboxCompleted(ctx context.Context, id int64) error
	MarkFulfillmentOutboxFailed(ctx context.Context, id int64, errMsg string) error
}

type SQLStore struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &SQLStore{
		db: db,
	}
}

// ==========================================
// Profile Update Outbox (用户画像更新 Outbox)
// ==========================================

func (s *SQLStore) PublishProfileUpdate(ctx context.Context, outbox *models.ProfileUpdateOutbox) error {
	return s.db.WithContext(ctx).Create(outbox).Error
}

func (s *SQLStore) GetPendingOutboxEntries(ctx context.Context, limit int) ([]models.ProfileUpdateOutbox, error) {
	var entries []models.ProfileUpdateOutbox
	err := s.db.WithContext(ctx).
		Where("status = ? AND retry_count < max_retries", models.OutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

func (s *SQLStore) MarkOutboxCompleted(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).
		Model(&models.ProfileUpdateOutbox{}).
		Where("id = ?", id).
		Update("status", models.OutboxStatusCompleted).Error
}

func (s *SQLStore) MarkOutboxFailed(ctx context.Context, id int64, errMsg string) error {
	return s.db.WithContext(ctx).
		Model(&models.ProfileUpdateOutbox{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.OutboxStatusFailed,
			"last_error":  errMsg,
			"retry_count": s.db.Raw("retry_count + 1"),
		}).Error
}

// ==========================================
// Bounties (悬赏操作)
// ==========================================
func (s *SQLStore) CreateBounty(ctx context.Context, bounty *models.Bounty) error {
	return s.db.WithContext(ctx).Create(bounty).Error
}

// 根据 ID 获取单个悬赏详情，并使用 Preload 预加载关联的申请列表
func (s *SQLStore) GetBountyByID(ctx context.Context, id int64) (*models.Bounty, error) {
	var bounty models.Bounty
	err := s.db.WithContext(ctx).Preload("Applications").First(&bounty, id).Error
	if err != nil {
		return nil, err
	}
	return &bounty, nil
}

// 分页获取悬赏列表，支持按状态过滤
func (s *SQLStore) ListBounties(ctx context.Context, status models.BountyStatus, limit, offset int) ([]models.Bounty, error) {
	var bounties []models.Bounty
	query := s.db.WithContext(ctx).Limit(limit).Offset(offset)

	// 如果传入了状态参数，则动态拼接 WHERE 条件
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&bounties).Error
	return bounties, err
}

func (s *SQLStore) DeleteBounty(ctx context.Context, bountyID int64, employerUsername string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		if err := tx.First(&bounty, bountyID).Error; err != nil {
			return err
		}

		if bounty.EmployerUsername != employerUsername {
			return fmt.Errorf("权限不足: 只能删除您自己发布的悬赏")
		}

		if bounty.Status != models.BountyStatusPending && bounty.Status != models.BountyStatusFailed {
			return fmt.Errorf("该悬赏当前状态 (%s) 无法删除，仅支持 PENDING/FAILED 状态的悬赏", bounty.Status)
		}

		// Delete associated applications first (CASCADE should handle this but be explicit)
		if err := tx.Where("bounty_id = ?", bountyID).Delete(&models.BountyApplication{}).Error; err != nil {
			return err
		}

		// Delete associated comments
		if err := tx.Where("bounty_id = ?", bountyID).Delete(&models.Comment{}).Error; err != nil {
			return err
		}

		return tx.Delete(&bounty).Error
	})
}

// UpdateBountyStatus 更新悬赏的状态
func (s *SQLStore) UpdateBountyStatus(ctx context.Context, id int64, status models.BountyStatus) error {
	return s.db.WithContext(ctx).
		Model(&models.Bounty{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// ==========================================
// Bounty Applications (接单申请操作)
// ==========================================

func (s *SQLStore) CreateApplication(ctx context.Context, app *models.BountyApplication) error {
	return s.db.WithContext(ctx).Create(app).Error
}

func (s *SQLStore) UpdateApplicationStatus(ctx context.Context, applicationID int64, status models.ApplicationStatus) error {
	return s.db.WithContext(ctx).
		Model(&models.BountyApplication{}).
		Where("id = ?", applicationID).
		Update("status", status).Error
}

// ==========================================
// rpc操作与业务逻辑
// ==========================================

type BankClient interface {
	Transfer(ctx context.Context, fromAccount, toAccount int64, amount int64, idempotencyKey string) error
	VerifyAccountOwner(ctx context.Context, accountID int64) error
	ListAccounts(ctx context.Context) ([]*simplebankpb.Account, error)
	Freeze(ctx context.Context, employerAccountID, amount, bountyID int64, description, idempotencyKey string) error
	Unfreeze(ctx context.Context, employerAccountID, amount, bountyID int64, description, idempotencyKey string) error
	BountyPayout(ctx context.Context, employerAccountID, hunterAccountID, amount, bountyID int64, description, idempotencyKey string) error
}

func (s *SQLStore) PublishBounty(ctx context.Context, bounty *models.Bounty, bankClient BankClient, employerAccountID int64) error {

	bounty.Status = models.BountyStatusPaying
	bounty.EmployerAccountID = employerAccountID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(bounty).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("本地创建悬赏失败: %w", err)
	}

	idempotencyKey := fmt.Sprintf("publish_bounty_%d", bounty.ID)

	// Freeze bounty amount internally in employer's account (no real transfer)
	rpcErr := bankClient.Freeze(ctx, employerAccountID, bounty.RewardAmount, bounty.ID,
		fmt.Sprintf("悬赏 #%d 冻结", bounty.ID), idempotencyKey)

	if rpcErr != nil {
		// 分布式系统的部分失败处理 (Partial Failure)
		log.Printf("调用 Simplebank 冻结异常: %v\n", rpcErr)

		updateErr := s.db.WithContext(ctx).Model(bounty).Update("status", models.BountyStatusFailed).Error
		if updateErr != nil {
			log.Printf("严重警告: 状态机回滚失败，bounty_id=%d, err=%v\n", bounty.ID, updateErr)
		}
		return fmt.Errorf("资金冻结失败，悬赏发布终止: %w", rpcErr)
	}

	// gRPC 明确返回成功，更新状态为 PENDING，任务正式进入悬赏大厅
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(bounty).Update("status", models.BountyStatusPending).Error; err != nil {
			return err
		}

		// 写入 outbox：雇主发布数 +1
		outbox := &models.ProfileUpdateOutbox{
			Username:                 bounty.EmployerUsername,
			BountyID:                bounty.ID,
			DeltaCompleted:          0,
			DeltaEarnings:           0,
			DeltaPosted:             1,
			DeltaCompletedAsEmployer: 0,
			Status:                  models.OutboxStatusPending,
			RequestID:               fmt.Sprintf("publish_bounty_%d", bounty.ID),
		}
		if err := tx.Create(outbox).Error; err != nil {
			return fmt.Errorf("写入 outbox 失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("资金已冻结，但更新本地状态为 PENDING 时出错: %w", err)
	}

	bounty.Status = models.BountyStatusPending
	return nil
}

// AcceptBounty 处理猎人“抢单/申请”逻辑
func (s *SQLStore) AcceptBounty(ctx context.Context, bountyID, hunter_account_id int64, hunterUsername string) (*models.BountyApplication, error) {
	var application models.BountyApplication

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bounty, bountyID).Error; err != nil {
			return err
		}

		if bounty.Status != models.BountyStatusPending {
			return fmt.Errorf("该悬赏当前不可接单，状态为: %s", bounty.Status)
		}

		application = models.BountyApplication{
			BountyID:        bountyID,
			HunterUsername:  hunterUsername,
			HunterAccountID: hunter_account_id,
			Status:          models.AppStatusApplied,
		}
		// 落库
		if err := tx.Create(&application).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (s *SQLStore) ConfirmHunter(ctx context.Context, bountyID int64, applicationID int64, employerUsername string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		// 加上 FOR UPDATE 锁，防止并发修改
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bounty, bountyID).Error; err != nil {
			return err
		}

		if bounty.EmployerUsername != employerUsername {
			return fmt.Errorf("权限不足: 只能操作您自己发布的悬赏")
		}
		if bounty.Status != models.BountyStatusPending {
			return fmt.Errorf("悬赏状态不合法，当前状态: %s", bounty.Status)
		}

		// 将选中的申请状态改为 ACCEPTED
		res := tx.Model(&models.BountyApplication{}).
			Where("id = ? AND bounty_id = ?", applicationID, bountyID).
			Update("status", models.AppStatusAccepted)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("找不到对应的申请记录")
		}

		// 将其他落选的申请状态改为 REJECTED
		if err := tx.Model(&models.BountyApplication{}).
			Where("bounty_id = ? AND id != ?", bountyID, applicationID).
			Update("status", models.AppStatusRejected).Error; err != nil {
			return err
		}

		// 将悬赏本身状态改为进行中
		return tx.Model(&bounty).Update("status", models.BountyStatusInProgress).Error
	})
}

func (s *SQLStore) CompleteBounty(ctx context.Context, bountyID int64, employerUsername string, bankClient BankClient) error {
	// Step 1: 查询 bounty 和猎人信息
	var bounty models.Bounty
	if err := s.db.WithContext(ctx).First(&bounty, bountyID).Error; err != nil {
		return fmt.Errorf("查询悬赏失败: %w", err)
	}

	if bounty.EmployerUsername != employerUsername {
		return fmt.Errorf("权限不足: 非悬赏发布者")
	}

	// Step 2: 幂等检查 - 已完成直接返回成功
	if bounty.Status == models.BountyStatusCompleted {
		return nil
	}

	// Step 3: 查询猎人账户信息
	var app models.BountyApplication
	if err := s.db.WithContext(ctx).
		Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusAccepted).
		First(&app).Error; err != nil {
		return fmt.Errorf("找不到中标的猎人记录: %w", err)
	}

	// Step 4: 乐观锁更新状态 → COMPLETED
	// 允许从 IN_PROGRESS 或 SETTLING 升级到 COMPLETED
	result := s.db.WithContext(ctx).Model(&models.Bounty{}).
		Where("id = ? AND status IN (?, ?)", bountyID, models.BountyStatusInProgress, "SETTLING").
		Update("status", models.BountyStatusCompleted)
	if result.Error != nil {
		return fmt.Errorf("更新悬赏状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("悬赏当前状态 (%s) 无法完成，仅支持进行中或结算中的悬赏", bounty.Status)
	}

	// Step 5: 写入 outbox 和履约记录（幂等，重复执行也安全）
	hunterOutbox := &models.ProfileUpdateOutbox{
		Username:                  app.HunterUsername,
		BountyID:                  bountyID,
		DeltaCompleted:             1,
		DeltaEarnings:             bounty.RewardAmount,
		DeltaPosted:                0,
		DeltaCompletedAsEmployer:   0,
		Status:                    models.OutboxStatusPending,
		RequestID:                 fmt.Sprintf("complete_bounty_hunter_%d", bountyID),
	}
	s.db.WithContext(ctx).Create(hunterOutbox)

	employerOutbox := &models.ProfileUpdateOutbox{
		Username:                  bounty.EmployerUsername,
		BountyID:                  bountyID,
		DeltaCompleted:             0,
		DeltaEarnings:              0,
		DeltaPosted:                0,
		DeltaCompletedAsEmployer:    1,
		Status:                    models.OutboxStatusPending,
		RequestID:                 fmt.Sprintf("complete_bounty_employer_%d", bountyID),
	}
	s.db.WithContext(ctx).Create(employerOutbox)

	// 履约记录（先删后写，幂等）
	s.writeTaskRecordsTx(s.db.WithContext(ctx), &bounty, app.HunterUsername, models.OutcomeCompleted, "COMPLETED")
	s.writeTaskRecordsTx(s.db.WithContext(ctx), &bounty, bounty.EmployerUsername, models.OutcomeCompleted, "COMPLETED")

	// 触发第一次履约指数重算（使用默认 rating=3）
	if err := s.PublishFulfillmentRecalcTx(s.db.WithContext(ctx), app.HunterUsername, string(models.RoleHunter), bountyID); err != nil {
		log.Printf("CompleteBounty: 发布猎人履约重算失败: %v\n", err)
	}
	if err := s.PublishFulfillmentRecalcTx(s.db.WithContext(ctx), bounty.EmployerUsername, string(models.RoleEmployer), bountyID); err != nil {
		log.Printf("CompleteBounty: 发布雇主履约重算失败: %v\n", err)
	}

	// Step 6: 执行 BountyPayout（幂等键保护）
	idempotencyKey := fmt.Sprintf("complete_bounty_%d", bountyID)
	rpcErr := bankClient.BountyPayout(context.Background(), bounty.EmployerAccountID, app.HunterAccountID, bounty.RewardAmount, bountyID,
		fmt.Sprintf("悬赏 #%d 完成打款", bountyID), idempotencyKey)

	if rpcErr != nil {
		// 幂等冲突 = 已打款过，视为成功
		if isIdempotencyError(rpcErr) {
			log.Printf("CompleteBounty: BountyPayout 已执行（幂等拦截），bountyID=%d\n", bountyID)
			return nil
		}
		// 真实失败：回滚状态为 IN_PROGRESS，允许用户重试
		log.Printf("CompleteBounty: BountyPayout 失败，回滚状态。bountyID=%d, err=%v\n", bountyID, rpcErr)
		s.db.WithContext(ctx).Model(&models.Bounty{}).
			Where("id = ? AND status = ?", bountyID, models.BountyStatusCompleted).
			Update("status", models.BountyStatusInProgress)
		return fmt.Errorf("资金打款到猎人账户失败: %w", rpcErr)
	}

	return nil
}

// SubmitBounty allows the hunter to submit their work for review.
// IN_PROGRESS -> SUBMITTED (支持首次提交和被拒绝后重新提交)
func (s *SQLStore) SubmitBounty(ctx context.Context, bountyID int64, hunterUsername, submissionText string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bounty, bountyID).Error; err != nil {
			return err
		}

		// Verify the hunter has an application (ACCEPTED 首次提交, APPLIED 重新提交)
		var app models.BountyApplication
		if err := tx.Where("bounty_id = ? AND hunter_username = ?",
			bountyID, hunterUsername).First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("您还没有申请此悬赏")
			}
			return err
		}

		// 如果申请状态不是 ACCEPTED 也不是 APPLIED，不允许提交
		if app.Status != models.AppStatusAccepted && app.Status != models.AppStatusApplied {
			return fmt.Errorf("当前申请状态为 %s，无法提交", app.Status)
		}

		// 悬赏状态必须是 IN_PROGRESS (正常流程) 或 REJECTED (修复前的旧数据)
		if bounty.Status != models.BountyStatusInProgress && bounty.Status != models.BountyStatusRejected {
			return fmt.Errorf("悬赏当前状态为 %s，无法提交工作", bounty.Status)
		}

		// 更新申请状态为 ACCEPTED，悬赏状态为 SUBMITTED
		if err := tx.Model(&app).Update("status", models.AppStatusAccepted).Error; err != nil {
			return err
		}

		return tx.Model(&bounty).Updates(map[string]interface{}{
			"status":          models.BountyStatusSubmitted,
			"submission_text": submissionText,
		}).Error
	})
}

// ApproveBounty allows the publisher to approve the hunter's submission.
// SUBMITTED -> COMPLETED (with payment to hunter)
func (s *SQLStore) ApproveBounty(ctx context.Context, bountyID int64, employerUsername string, bankClient BankClient) error {
	// Step 1: 查询 bounty
	var bounty models.Bounty
	if err := s.db.WithContext(ctx).First(&bounty, bountyID).Error; err != nil {
		return fmt.Errorf("查询悬赏失败: %w", err)
	}

	if bounty.EmployerUsername != employerUsername {
		return fmt.Errorf("权限不足: 非悬赏发布者")
	}

	// Step 2: 幂等检查 - 已完成直接返回成功
	if bounty.Status == models.BountyStatusCompleted {
		return nil
	}

	// Step 3: 查询猎人账户信息
	var app models.BountyApplication
	if err := s.db.WithContext(ctx).
		Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusAccepted).
		First(&app).Error; err != nil {
		return fmt.Errorf("找不到中标的猎人记录: %w", err)
	}

	// Step 4: 乐观锁更新状态 SUBMITTED → COMPLETED
	result := s.db.WithContext(ctx).Model(&models.Bounty{}).
		Where("id = ? AND status = ?", bountyID, models.BountyStatusSubmitted).
		Update("status", models.BountyStatusCompleted)
	if result.Error != nil {
		return fmt.Errorf("更新悬赏状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("悬赏当前状态为 %s，无法通过，仅限已提交的悬赏", bounty.Status)
	}

	// Step 5: 写入 outbox 和履约记录（幂等，重复执行也安全）
	hunterOutbox := &models.ProfileUpdateOutbox{
		Username:                  app.HunterUsername,
		BountyID:                  bountyID,
		DeltaCompleted:             1,
		DeltaEarnings:             bounty.RewardAmount,
		DeltaPosted:                0,
		DeltaCompletedAsEmployer:    0,
		Status:                    models.OutboxStatusPending,
		RequestID:                 fmt.Sprintf("approve_bounty_hunter_%d", bountyID),
	}
	s.db.WithContext(ctx).Create(hunterOutbox)

	employerOutbox := &models.ProfileUpdateOutbox{
		Username:                  bounty.EmployerUsername,
		BountyID:                  bountyID,
		DeltaCompleted:             0,
		DeltaEarnings:              0,
		DeltaPosted:                0,
		DeltaCompletedAsEmployer:   1,
		Status:                    models.OutboxStatusPending,
		RequestID:                 fmt.Sprintf("approve_bounty_employer_%d", bountyID),
	}
	s.db.WithContext(ctx).Create(employerOutbox)

	// 履约记录（先删后写，幂等）
	s.writeTaskRecordsTx(s.db.WithContext(ctx), &bounty, app.HunterUsername, models.OutcomeCompleted, "COMPLETED")
	s.writeTaskRecordsTx(s.db.WithContext(ctx), &bounty, bounty.EmployerUsername, models.OutcomeCompleted, "COMPLETED")

	// 触发第一次履约指数重算（使用默认 rating=3）
	if err := s.PublishFulfillmentRecalcTx(s.db.WithContext(ctx), app.HunterUsername, string(models.RoleHunter), bountyID); err != nil {
		log.Printf("ApproveBounty: 发布猎人履约重算失败: %v\n", err)
	}
	if err := s.PublishFulfillmentRecalcTx(s.db.WithContext(ctx), bounty.EmployerUsername, string(models.RoleEmployer), bountyID); err != nil {
		log.Printf("ApproveBounty: 发布雇主履约重算失败: %v\n", err)
	}

	// Step 6: 执行 BountyPayout（幂等键保护）
	idempotencyKey := fmt.Sprintf("approve_bounty_%d", bountyID)
	rpcErr := bankClient.BountyPayout(context.Background(), bounty.EmployerAccountID, app.HunterAccountID, bounty.RewardAmount, bountyID,
		fmt.Sprintf("悬赏 #%d 完成打款", bountyID), idempotencyKey)

	if rpcErr != nil {
		// 幂等冲突 = 已打款过，视为成功
		if isIdempotencyError(rpcErr) {
			log.Printf("ApproveBounty: BountyPayout 已执行（幂等拦截），bountyID=%d\n", bountyID)
			return nil
		}
		// 真实失败：回滚状态为 SUBMITTED，允许用户重试
		log.Printf("ApproveBounty: BountyPayout 失败，回滚状态。bountyID=%d, err=%v\n", bountyID, rpcErr)
		s.db.WithContext(ctx).Model(&models.Bounty{}).
			Where("id = ? AND status = ?", bountyID, models.BountyStatusCompleted).
			Update("status", models.BountyStatusSubmitted)
		return fmt.Errorf("资金打款到猎人账户失败: %w", rpcErr)
	}

	return nil
}

// RejectBounty allows the publisher to reject the hunter's submission.
// SUBMITTED -> IN_PROGRESS (猎人可再次提交)
func (s *SQLStore) RejectBounty(ctx context.Context, bountyID int64, employerUsername string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bounty, bountyID).Error; err != nil {
			return err
		}

		if bounty.EmployerUsername != employerUsername {
			return fmt.Errorf("权限不足: 非悬赏发布者")
		}
		if bounty.Status != models.BountyStatusSubmitted {
			return fmt.Errorf("悬赏当前状态为 %s，无法拒绝，仅限已提交的悬赏", bounty.Status)
		}

		// 找到中标猎人
		var app models.BountyApplication
		if err := tx.Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusAccepted).First(&app).Error; err != nil {
			return fmt.Errorf("找不到中标的猎人记录: %w", err)
		}
		hunterUsername := app.HunterUsername

		// 重置application状态为APPLIED，允许猎人重新提交（不退款）
		if err := tx.Model(&models.BountyApplication{}).
			Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusAccepted).
			Update("status", models.AppStatusApplied).Error; err != nil {
			return err
		}

		// 写入履约记录：猎人和雇主各一条，Outcome=0（中性，不扣分）
		if err := s.writeTaskRecordsTx(tx, &bounty, hunterUsername, models.OutcomeNeutral, "REJECTED"); err != nil {
			return fmt.Errorf("写入猎人履约记录失败: %w", err)
		}
		if err := s.writeTaskRecordsTx(tx, &bounty, employerUsername, models.OutcomeNeutral, "REJECTED"); err != nil {
			return fmt.Errorf("写入雇主履约记录失败: %w", err)
		}

		// 触发履约指数重算（outbox 模式）
		if err := s.PublishFulfillmentRecalcTx(tx, hunterUsername, string(models.RoleHunter), bounty.ID); err != nil {
			log.Printf("写入猎人履约重算 outbox 失败: %v\n", err)
		}
		if err := s.PublishFulfillmentRecalcTx(tx, employerUsername, string(models.RoleEmployer), bounty.ID); err != nil {
			log.Printf("写入雇主履约重算 outbox 失败: %v\n", err)
		}

		// 状态改回 IN_PROGRESS，让猎人可以再次提交
		return tx.Model(&bounty).Update("status", models.BountyStatusInProgress).Error
	})
}

// ReSubmitBounty allows the hunter to resubmit after rejection.
// IN_PROGRESS -> SUBMITTED (reject 把状态改回 IN_PROGRESS，猎人可再次提交)
func (s *SQLStore) ReSubmitBounty(ctx context.Context, bountyID int64, hunterUsername, submissionText string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bounty, bountyID).Error; err != nil {
			return err
		}

		// Verify the hunter has an APPLIED application (被拒绝后会变成 APPLIED)
		var app models.BountyApplication
		if err := tx.Where("bounty_id = ? AND status = ? AND hunter_username = ?",
			bountyID, models.AppStatusApplied, hunterUsername).First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("找不到对应的申请记录")
			}
			return err
		}

		if bounty.Status != models.BountyStatusInProgress {
			return fmt.Errorf("悬赏当前状态为 %s，无法重新提交，仅限进行中的悬赏", bounty.Status)
		}

		// 更新申请状态为 ACCEPTED
		if err := tx.Model(&app).Update("status", models.AppStatusAccepted).Error; err != nil {
			return err
		}

		return tx.Model(&bounty).Updates(map[string]interface{}{
			"status":          models.BountyStatusSubmitted,
			"submission_text": submissionText,
		}).Error
	})
}

// CancelBounty 取消悬赏（带乐观锁 + 幂等性 + 失败回滚）
func (s *SQLStore) CancelBounty(ctx context.Context, bountyID int64, employerUsername string, bankClient BankClient) error {
	// Step 1: 查询 bounty 基础信息
	var bounty models.Bounty
	if err := s.db.WithContext(ctx).First(&bounty, bountyID).Error; err != nil {
		return fmt.Errorf("查询悬赏失败: %w", err)
	}

	if bounty.EmployerUsername != employerUsername {
		return fmt.Errorf("权限不足: 只能取消您自己发布的悬赏")
	}

	// Step 2: 幂等检查 - 已取消直接返回成功
	if bounty.Status == models.BountyStatusCanceled {
		return nil
	}

	// Step 3: 乐观锁更新状态 PENDING → CANCELED
	// 使用 UPDATE ... WHERE status = 'PENDING' 替代 FOR UPDATE，避免长时间锁
	result := s.db.WithContext(ctx).Model(&models.Bounty{}).
		Where("id = ? AND status = ?", bountyID, models.BountyStatusPending).
		Update("status", models.BountyStatusCanceled)
	if result.Error != nil {
		return fmt.Errorf("更新悬赏状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("该悬赏当前状态 (%s) 无法取消，仅支持未开始的任务", bounty.Status)
	}

	// Step 4: 清理关联数据（申请变 REJECTED，履约记录）
	// 这些是幂等的清理操作，即使重复执行也安全
	s.db.WithContext(ctx).Model(&models.BountyApplication{}).
		Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusApplied).
		Update("status", models.AppStatusRejected)

	// 写入履约记录（幂等：先删后写）
	role := models.RoleEmployer
	s.db.WithContext(ctx).Where("bounty_id = ? AND username = ? AND role = ?", bountyID, employerUsername, role).
		Delete(&models.TaskRecord{})
	task := models.TaskRecord{
		Username:        employerUsername,
		Role:            role,
		BountyID:        bountyID,
		Amount:          bounty.RewardAmount,
		Outcome:         models.OutcomeNeutral,
		OutcomeDetail:   "CANCELED",
		EmployerRating:  3,
		HunterRating:   3,
	}
	s.db.WithContext(ctx).Create(&task)

	// Step 5: 执行 Unfreeze（幂等键保护）
	idempotencyKey := fmt.Sprintf("cancel_bounty_refund_%d", bountyID)
	rpcErr := bankClient.Unfreeze(context.Background(), bounty.EmployerAccountID, bounty.RewardAmount, bountyID,
		fmt.Sprintf("悬赏 #%d 取消退款", bountyID), idempotencyKey)

	if rpcErr != nil {
		// 检查是否是幂等冲突（已经退过款了）
		if isIdempotencyError(rpcErr) {
			log.Printf("CancelBounty: Unfreeze 已执行（幂等拦截），bountyID=%d\n", bountyID)
			return nil
		}
		// 真实失败：回滚状态为 PENDING，允许用户重试
		log.Printf("CancelBounty: Unfreeze 失败，回滚状态。bountyID=%d, err=%v\n", bountyID, rpcErr)
		s.db.WithContext(ctx).Model(&models.Bounty{}).
			Where("id = ? AND status = ?", bountyID, models.BountyStatusCanceled).
			Update("status", models.BountyStatusPending)
		return fmt.Errorf("取消悬赏失败，解冻资金失败: %w", rpcErr)
	}

	return nil
}

// isIdempotencyError 检查 RPC 错误是否是幂等性键冲突
func isIdempotencyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "idempotency") ||
		strings.Contains(errStr, "AlreadyExists") ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "ALREADY_EXISTS")
}

// ==========================================
// Chat (聊天操作)
// ==========================================

func (s *SQLStore) GetOrCreateChat(ctx context.Context, bountyID int64) (*models.Chat, error) {
	var chat models.Chat
	err := s.db.WithContext(ctx).
		Where("bounty_id = ?", bountyID).
		First(&chat).Error
	if err == nil {
		return &chat, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Lazily create the chat row
	var bounty models.Bounty
	if err := s.db.WithContext(ctx).First(&bounty, bountyID).Error; err != nil {
		return nil, err
	}
	var acceptedApp models.BountyApplication
	if err := s.db.WithContext(ctx).
		Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusAccepted).
		First(&acceptedApp).Error; err != nil {
		return nil, fmt.Errorf("no accepted hunter for bounty %d: %w", bountyID, err)
	}

	chat = models.Chat{
		BountyID:         bountyID,
		EmployerUsername: bounty.EmployerUsername,
		HunterUsername:   acceptedApp.HunterUsername,
	}
	if err := s.db.WithContext(ctx).Create(&chat).Error; err != nil {
		// Handle race: another goroutine may have inserted first
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "SQLSTATE 23505") {
			_ = s.db.WithContext(ctx).
				Where("bounty_id = ?", bountyID).
				First(&chat).Error
			return &chat, nil
		}
		return nil, err
	}
	return &chat, nil
}

func (s *SQLStore) CreateMessage(ctx context.Context, chatID int64, senderUsername, content string) (*models.ChatMessage, error) {
	msg := &models.ChatMessage{
		ChatID:         chatID,
		SenderUsername: senderUsername,
		Content:        content,
	}
	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return nil, err
	}
	// Touch updated_at on the parent chat
	s.db.WithContext(ctx).Model(&models.Chat{}).Where("id = ?", chatID).Update("updated_at", time.Now())
	return msg, nil
}

func (s *SQLStore) ListMessages(ctx context.Context, bountyID int64) ([]models.ChatMessage, error) {
	var chat models.Chat
	if err := s.db.WithContext(ctx).Where("bounty_id = ?", bountyID).First(&chat).Error; err != nil {
		return nil, err
	}
	var msgs []models.ChatMessage
	err := s.db.WithContext(ctx).
		Where("chat_id = ?", chat.ID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// ==========================================
// Private Chat (私信操作)
// ==========================================

func (s *SQLStore) GetOrCreatePrivateConversation(ctx context.Context, user1, user2 string) (*models.PrivateConversation, error) {
	// Alphabetically order to ensure unique constraint works
	if user1 > user2 {
		user1, user2 = user2, user1
	}

	var conv models.PrivateConversation
	err := s.db.WithContext(ctx).
		Where("user1_username = ? AND user2_username = ?", user1, user2).
		First(&conv).Error
	if err == nil {
		return &conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	conv = models.PrivateConversation{
		User1Username: user1,
		User2Username: user2,
	}
	if err := s.db.WithContext(ctx).Create(&conv).Error; err != nil {
		// Handle race condition
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "SQLSTATE 23505") {
			_ = s.db.WithContext(ctx).
				Where("user1_username = ? AND user2_username = ?", user1, user2).
				First(&conv).Error
			return &conv, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (s *SQLStore) ListPrivateConversations(ctx context.Context, username string) ([]models.PrivateConversation, error) {
	var convs []models.PrivateConversation
	err := s.db.WithContext(ctx).
		Where("user1_username = ? OR user2_username = ?", username, username).
		Order("updated_at DESC").
		Find(&convs).Error
	return convs, err
}

func (s *SQLStore) DeletePrivateConversation(ctx context.Context, convID int64, username string) error {
	var conv models.PrivateConversation
	if err := s.db.WithContext(ctx).First(&conv, convID).Error; err != nil {
		return err
	}
	// Verify caller is a participant
	if conv.User1Username != username && conv.User2Username != username {
		return fmt.Errorf("permission denied")
	}
	return s.db.WithContext(ctx).Delete(&conv).Error
}

// ==========================================
// Unified Messages (统一消息表)
// ==========================================

func (s *SQLStore) CreateMessageV2(ctx context.Context, msgType string, convID, bountyChatID *int64, senderUsername, content string) (*models.AllMessage, error) {
	msg := &models.AllMessage{
		MessageType:    msgType,
		ConversationID: convID,
		BountyChatID:   bountyChatID,
		SenderUsername: senderUsername,
		Content:        content,
		IsRead:         false,
	}
	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return nil, err
	}

	// Update conversation updated_at
	if convID != nil {
		s.db.WithContext(ctx).Model(&models.PrivateConversation{}).Where("id = ?", *convID).Update("updated_at", time.Now())
	} else if bountyChatID != nil {
		s.db.WithContext(ctx).Model(&models.Chat{}).Where("id = ?", *bountyChatID).Update("updated_at", time.Now())
	}

	return msg, nil
}

func (s *SQLStore) ListMessagesV2(ctx context.Context, convID int64, limit, offset int) ([]models.AllMessage, error) {
	var msgs []models.AllMessage
	err := s.db.WithContext(ctx).
		Where("conversation_id = ?", convID).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&msgs).Error
	return msgs, err
}

func (s *SQLStore) ListBountyMessagesV2(ctx context.Context, bountyChatID int64, limit, offset int) ([]models.AllMessage, error) {
	var msgs []models.AllMessage
	err := s.db.WithContext(ctx).
		Where("bounty_chat_id = ?", bountyChatID).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&msgs).Error
	return msgs, err
}

func (s *SQLStore) MarkMessagesRead(ctx context.Context, convID int64, readerUsername string) error {
	return s.db.WithContext(ctx).
		Model(&models.AllMessage{}).
		Where("conversation_id = ? AND sender_username != ? AND is_read = ?", convID, readerUsername, false).
		Update("is_read", true).Error
}

func (s *SQLStore) GetUnreadCounts(ctx context.Context, username string) (map[int64]int, error) {
	var counts []models.ConversationUnreadCount
	err := s.db.WithContext(ctx).
		Where("username = ? AND unread_count > 0", username).
		Find(&counts).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int)
	for _, c := range counts {
		result[c.ConversationID] = c.UnreadCount
	}
	return result, nil
}

// ==========================================
// Comments (评论区)
// ==========================================

func (s *SQLStore) ListComments(ctx context.Context, bountyID int64) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.WithContext(ctx).
		Where("bounty_id = ?", bountyID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (s *SQLStore) GetAvatarUrlsByUsernames(ctx context.Context, usernames []string) (map[string]string, error) {
	if len(usernames) == 0 {
		return map[string]string{}, nil
	}
	type result struct {
		Username  string
		AvatarURL string
	}
	var results []result
	err := s.db.WithContext(ctx).
		Raw("SELECT username, COALESCE(avatar_url, '') AS avatar_url FROM user_profiles WHERE username = ANY($1)", pq.Array(usernames)).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(results))
	for _, r := range results {
		m[r.Username] = r.AvatarURL
	}
	return m, nil
}

func (s *SQLStore) GetComment(ctx context.Context, id int64) (*models.Comment, error) {
	var comment models.Comment
	if err := s.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (s *SQLStore) CreateComment(ctx context.Context, bountyID int64, replyToID *int64, authorUsername, content string, imageID *int64) (*models.Comment, error) {
	// Verify bounty exists
	var bounty models.Bounty
	if err := s.db.WithContext(ctx).First(&bounty, bountyID).Error; err != nil {
		return nil, err
	}

	var parentID *int64
	// 规则 1: 没有 replyToID -> 这是根评论，parentID = NULL
	// 规则 2: 有 replyToID，检测其 parent_id 是否为空
	//   - 为空 -> replyToID 就是父亲，parentID = replyToID
	//   - 非空 -> 继承父亲，parentID = replyToID 的 parent_id，replyToID 保持不变
	if replyToID != nil {
		var replyTo models.Comment
		if err := s.db.WithContext(ctx).First(&replyTo, *replyToID).Error; err != nil {
			return nil, err
		}
		if replyTo.BountyID != bountyID {
			return nil, fmt.Errorf("parent comment does not belong to this bounty")
		}
		if replyTo.ParentID == nil {
			// 规则 2: replyTo 本身是父亲评论
			parentID = replyToID
		} else {
			// 规则 3: 继承父亲评论的 parent_id
			parentID = replyTo.ParentID
		}
	}

	comment := &models.Comment{
		BountyID:       bountyID,
		ParentID:       parentID,  // NULL 表示根评论，非 NULL 指向根评论
		ReplyToID:      replyToID, // NULL 表示直接回复父亲，非 NULL 指向具体被回复的评论
		AuthorUsername: authorUsername,
		Content:        content,
		ImageID:        imageID,
	}
	if err := s.db.WithContext(ctx).Create(comment).Error; err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *SQLStore) DeleteComment(ctx context.Context, commentID int64, username string) error {
	var comment models.Comment
	if err := s.db.WithContext(ctx).First(&comment, commentID).Error; err != nil {
		return err
	}
	if comment.AuthorUsername != username {
		return fmt.Errorf("permission denied: can only delete your own comments")
	}
	// Cascade delete child comments (replies)
	if err := s.db.WithContext(ctx).Where("parent_id = ?", commentID).Delete(&models.Comment{}).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Delete(&comment).Error
}

// DeleteCommentCascade 删除评论 + 所有子孙评论（通过 reply_to_id 链向上追溯）+ 关联图片
// 返回被删除的图片相对路径列表，由调用方负责物理文件清理
func (s *SQLStore) DeleteCommentCascade(ctx context.Context, commentID int64, username string) ([]string, error) {
	var deletedPaths []string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var comment models.Comment
		if err := tx.First(&comment, commentID).Error; err != nil {
			return err
		}
		if comment.AuthorUsername != username {
			return fmt.Errorf("permission denied: can only delete your own comments")
		}

		// 收集所有要删除的评论ID
		allIDs := []int64{commentID}

		// 递归收集所有 reply_to_id 指向已删除评论的子孙评论
		queue := []int64{commentID}
		for len(queue) > 0 {
			currentID := queue[0]
			queue = queue[1:]
			var childReplyIDs []int64
			tx.Model(&models.Comment{}).Where("reply_to_id = ?", currentID).Pluck("id", &childReplyIDs)
			for _, childID := range childReplyIDs {
				allIDs = append(allIDs, childID)
				queue = append(queue, childID)
			}
		}

		// 查询关联图片路径（用于物理清理）
		// 图片通过 comments.image_id 关联，按 image_id 查
		var imgIDs []int64
		tx.Model(&models.Comment{}).Where("id IN ?", allIDs).Pluck("COALESCE(image_id, 0)", &imgIDs)
		realImgIDs := make([]int64, 0)
		for _, id := range imgIDs {
			if id > 0 {
				realImgIDs = append(realImgIDs, id)
			}
		}
		var imgs []models.Image
		if len(realImgIDs) > 0 {
			tx.Where("id IN ?", realImgIDs).Find(&imgs)
			for _, img := range imgs {
				deletedPaths = append(deletedPaths, img.RelativePath)
			}
			tx.Where("id IN ?", realImgIDs).Delete(&models.Image{})
		}

		// 删除所有子孙评论（通过 reply_to_id 链找到的）
		if len(allIDs) > 1 {
			if err := tx.Where("id IN ?", allIDs[1:]).Delete(&models.Comment{}).Error; err != nil {
				return err
			}
		}

		// 删除自己
		return tx.Delete(&comment).Error
	})
	return deletedPaths, err
}

func (s *SQLStore) ListCommentsByUsername(ctx context.Context, username string, limit, offset int) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.WithContext(ctx).
		Where("author_username = ?", username).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error
	return comments, err
}

// ==========================================
// Images (统一图片资源)
// ==========================================

func (s *SQLStore) CreateImage(ctx context.Context, img *models.Image) error {
	return s.db.WithContext(ctx).Create(img).Error
}

func (s *SQLStore) GetImage(ctx context.Context, id int64) (*models.Image, error) {
	var img models.Image
	if err := s.db.WithContext(ctx).First(&img, id).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

func (s *SQLStore) DeleteImage(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&models.Image{}, id).Error
}

func (s *SQLStore) ListImagesByEntity(ctx context.Context, entityType string, entityID string) ([]models.Image, error) {
	var imgs []models.Image
	err := s.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at ASC").
		Find(&imgs).Error
	return imgs, err
}

func (s *SQLStore) DeleteCommentImages(ctx context.Context, commentID int64) ([]models.Image, error) {
	var imgs []models.Image
	if err := s.db.WithContext(ctx).
		Where("entity_type = 'comment' AND entity_id = ?", commentID).
		Find(&imgs).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).
		Where("entity_type = 'comment' AND entity_id = ?", commentID).
		Delete(&models.Image{}).Error; err != nil {
		return nil, err
	}
	return imgs, nil
}

// ==========================================
// Invitations (邀请接单)
// ==========================================

func (s *SQLStore) CreateInvitation(ctx context.Context, inv *models.Invitation) (*models.Invitation, error) {
	err := s.db.WithContext(ctx).Create(inv).Error
	return inv, err
}

func (s *SQLStore) GetInvitationByID(ctx context.Context, id int64) (*models.Invitation, error) {
	var inv models.Invitation
	err := s.db.WithContext(ctx).Preload("Bounty").First(&inv, id).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (s *SQLStore) ListInvitationsByHunter(ctx context.Context, hunterUsername string, status models.InvitationStatus, limit, offset int) ([]models.Invitation, error) {
	var invs []models.Invitation
	query := s.db.WithContext(ctx).
		Preload("Bounty").
		Where("hunter_username = ?", hunterUsername).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&invs).Error
	return invs, err
}

func (s *SQLStore) ListInvitationsByPoster(ctx context.Context, posterUsername string, bountyID *int64, status models.InvitationStatus, limit, offset int) ([]models.Invitation, error) {
	var invs []models.Invitation
	query := s.db.WithContext(ctx).
		Preload("Bounty").
		Where("poster_username = ?", posterUsername).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)
	if bountyID != nil {
		query = query.Where("bounty_id = ?", *bountyID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&invs).Error
	return invs, err
}

func (s *SQLStore) UpdateInvitationStatus(ctx context.Context, id int64, status models.InvitationStatus) (*models.Invitation, error) {
	err := s.db.WithContext(ctx).
		Model(&models.Invitation{}).
		Where("id = ?", id).
		Update("status", status).Error
	if err != nil {
		return nil, err
	}
	return s.GetInvitationByID(ctx, id)
}

func (s *SQLStore) DeleteInvitation(ctx context.Context, id int64, posterUsername string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv models.Invitation
		if err := tx.First(&inv, id).Error; err != nil {
			return err
		}
		if inv.PosterUsername != posterUsername {
			return fmt.Errorf("permission denied: only poster can delete invitation")
		}
		if inv.Status != models.InvitationStatusPending {
			return fmt.Errorf("can only delete pending invitations")
		}
		return tx.Delete(&inv).Error
	})
}

func (s *SQLStore) ListApplicationsByHunter(ctx context.Context, hunterUsername string, status models.ApplicationStatus, limit, offset int) ([]models.BountyApplication, error) {
	var apps []models.BountyApplication
	if limit <= 0 {
		limit = 20
	}
	db := s.db.WithContext(ctx).Where("hunter_username = ?", hunterUsername)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&apps).Error
	return apps, err
}

// ==========================================
// Task Records (履约指数记录)
// ==========================================

// writeTaskRecordsTx 在事务中写入任务履约记录
// 方案A：写入新记录前，先删除该用户该悬赏的旧记录，确保每个 (bounty_id, username, role) 组合只有一条最新记录
func (s *SQLStore) writeTaskRecordsTx(tx *gorm.DB, bounty *models.Bounty, username string, outcome models.TaskRecordOutcome, outcomeDetail string) error {
	role := models.RoleEmployer
	if username != bounty.EmployerUsername {
		role = models.RoleHunter
	}

	// 删除该用户该悬赏的旧记录（确保只有一条）
	if err := tx.Where("bounty_id = ? AND username = ? AND role = ?", bounty.ID, username, role).
		Delete(&models.TaskRecord{}).Error; err != nil {
		return err
	}

	record := &models.TaskRecord{
		Username:        username,
		Role:            role,
		BountyID:        bounty.ID,
		Amount:          bounty.RewardAmount,
		Outcome:         outcome,
		OutcomeDetail:   outcomeDetail,
		EmployerRating: 3, // 默认 3 星，等待互评结算
		HunterRating:   3,
		DeadlineBefore: bounty.Deadline,
		ExtendCount:    bounty.ExtendCount,
		RatingFinalized: false,
	}
	return tx.Create(record).Error
}

func (s *SQLStore) WriteTaskRecords(ctx context.Context, bounty *models.Bounty, hunterUsername string, outcome models.TaskRecordOutcome, outcomeDetail string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.writeTaskRecordsTx(tx, bounty, hunterUsername, outcome, outcomeDetail); err != nil {
			return err
		}
		return s.writeTaskRecordsTx(tx, bounty, bounty.EmployerUsername, outcome, outcomeDetail)
	})
}

// ListExpiredBounties 查找已过期但仍在 IN_PROGRESS 状态的悬赏（用于定时任务）
func (s *SQLStore) ListExpiredBounties(ctx context.Context, limit int) ([]models.Bounty, error) {
	var bounties []models.Bounty
	err := s.db.WithContext(ctx).
		Where("status = ? AND deadline IS NOT NULL AND deadline < ?", models.BountyStatusInProgress, time.Now()).
		Limit(limit).
		Find(&bounties).Error
	return bounties, err
}

// ExpireBounty 将过期悬赏标记为 EXPIRED，并写入履约记录
func (s *SQLStore) ExpireBounty(ctx context.Context, bountyID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bounty models.Bounty
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bounty, bountyID).Error; err != nil {
			return err
		}

		if bounty.Status != models.BountyStatusInProgress {
			return nil // 已被处理，跳过
		}
		if bounty.Deadline != nil && bounty.Deadline.After(time.Now()) {
			return nil // 未过期，跳过
		}

		// 找到中标猎人
		var app models.BountyApplication
		if err := tx.Where("bounty_id = ? AND status = ?", bountyID, models.AppStatusAccepted).First(&app).Error; err != nil {
			return fmt.Errorf("找不到中标的猎人记录: %w", err)
		}
		hunterUsername := app.HunterUsername

		// 更新状态为 EXPIRED
		if err := tx.Model(&bounty).Update("status", models.BountyStatusExpired).Error; err != nil {
			return err
		}

		// 写入履约记录：猎人和雇主各一条，Outcome=-1（严重违约）
		if err := s.writeTaskRecordsTx(tx, &bounty, hunterUsername, models.OutcomeExpired, "DEADLINE_MISSED"); err != nil {
			return fmt.Errorf("写入猎人履约记录失败: %w", err)
		}
		if err := s.writeTaskRecordsTx(tx, &bounty, bounty.EmployerUsername, models.OutcomeExpired, "DEADLINE_MISSED"); err != nil {
			return fmt.Errorf("写入雇主履约记录失败: %w", err)
		}

		// 触发履约指数重算（outbox 模式）
		if err := s.PublishFulfillmentRecalcTx(tx, hunterUsername, string(models.RoleHunter), bounty.ID); err != nil {
			log.Printf("写入猎人履约重算 outbox 失败: %v\n", err)
		}
		if err := s.PublishFulfillmentRecalcTx(tx, bounty.EmployerUsername, string(models.RoleEmployer), bounty.ID); err != nil {
			log.Printf("写入雇主履约重算 outbox 失败: %v\n", err)
		}

		return nil
	})
}

// ==========================================
// Fulfillment Outbox (履约指数重算事件 Outbox)
// ==========================================

func (s *SQLStore) PublishFulfillmentRecalc(ctx context.Context, username, role string, bountyID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.PublishFulfillmentRecalcTx(tx, username, role, bountyID)
	})
}

func (s *SQLStore) PublishFulfillmentRecalcTx(tx *gorm.DB, username, role string, bountyID int64) error {
	outbox := &models.FulfillmentOutbox{
		Username:  username,
		Role:      role,
		BountyID:  bountyID,
		Status:    models.FulfillmentOutboxStatusPending,
		RequestID: fmt.Sprintf("fulfillment_%s_%s_%d", username, role, bountyID),
	}
	return tx.Create(outbox).Error
}

func (s *SQLStore) GetPendingFulfillmentOutbox(ctx context.Context, limit int) ([]models.FulfillmentOutbox, error) {
	var entries []models.FulfillmentOutbox
	err := s.db.WithContext(ctx).
		Where("status = ? AND retry_count < max_retries", models.FulfillmentOutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

func (s *SQLStore) MarkFulfillmentOutboxCompleted(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).
		Model(&models.FulfillmentOutbox{}).
		Where("id = ?", id).
		Update("status", models.FulfillmentOutboxStatusCompleted).Error
}

func (s *SQLStore) MarkFulfillmentOutboxFailed(ctx context.Context, id int64, errMsg string) error {
	return s.db.WithContext(ctx).
		Model(&models.FulfillmentOutbox{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.FulfillmentOutboxStatusFailed,
			"last_error":  errMsg,
			"retry_count": s.db.Raw("retry_count + 1"),
		}).Error
}


