package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	InvitationStatusPending = "pending"
	InvitationStatusUsed    = "used"
	InvitationStatusExpired = "expired"
	InvitationStatusRevoked = "revoked"

	invitationTokenBytes = 32
)

var InvitationStatuses = []string{InvitationStatusPending, InvitationStatusUsed, InvitationStatusExpired, InvitationStatusRevoked}

var (
	// ErrInvitationInvalid is the one answer for a token that is unknown, used,
	// revoked or expired, so a caller learns nothing about which.
	ErrInvitationInvalid  = errors.New("invitation is invalid or has expired")
	ErrInvitationPending  = errors.New("an invitation to this email address is still pending")
	ErrInvitationClosed   = errors.New("invitation has been used or revoked")
	ErrInvitationNotFound = errors.New("invitation not found")
)

// Invitation lets one person create an account with the invited email
// address. Only the SHA-256 of the mailed token is stored.
type Invitation struct {
	Id          int    `json:"id"`
	Email       string `json:"email" gorm:"index" validate:"max=50"`
	TokenHash   string `json:"-" gorm:"type:char(64);uniqueIndex"`
	Language    string `json:"language" gorm:"type:varchar(16)"`
	CreatedBy   int    `json:"created_by"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	ExpiredTime int64  `json:"expired_time" gorm:"bigint"`
	UsedTime    int64  `json:"used_time" gorm:"bigint"`
	UsedUserId  int    `json:"used_user_id"`
	RevokedTime int64  `json:"revoked_time" gorm:"bigint"`
	Status      string `json:"status" gorm:"-:all"`
}

func hashInvitationToken(token string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
}

func newInvitationToken() (string, error) {
	random := make([]byte, invitationTokenBytes)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate invitation token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}

// pendingInvitations narrows tx to invitations that are not used, not revoked
// and not expired at now.
func pendingInvitations(tx *gorm.DB, now int64) *gorm.DB {
	return tx.Where("used_time = 0 AND revoked_time = 0 AND expired_time > ?", now)
}

func (invitation *Invitation) fillStatus(now int64) {
	switch {
	case invitation.UsedTime > 0:
		invitation.Status = InvitationStatusUsed
	case invitation.RevokedTime > 0:
		invitation.Status = InvitationStatusRevoked
	case invitation.ExpiredTime <= now:
		invitation.Status = InvitationStatusExpired
	default:
		invitation.Status = InvitationStatusPending
	}
}

// CreateInvitation stores a pending invitation that expires validFor seconds
// from now and returns it with its token, which is not stored.
func CreateInvitation(email string, language string, createdBy int, validFor int64) (*Invitation, string, error) {
	token, err := newInvitationToken()
	if err != nil {
		return nil, "", err
	}
	now := common.GetTimestamp()
	invitation := &Invitation{
		Email:       NormalizeEmail(email),
		TokenHash:   hashInvitationToken(token),
		Language:    language,
		CreatedBy:   createdBy,
		CreatedTime: now,
		ExpiredTime: now + validFor,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var pending int64
		if err := pendingInvitations(tx.Model(&Invitation{}), now).Where("email = ?", invitation.Email).Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return ErrInvitationPending
		}
		return tx.Create(invitation).Error
	})
	if err != nil {
		return nil, "", err
	}
	invitation.fillStatus(now)
	return invitation, token, nil
}

// GetPendingInvitationByToken returns the pending invitation the token opens,
// or ErrInvitationInvalid.
func GetPendingInvitationByToken(token string) (*Invitation, error) {
	if token == "" {
		return nil, ErrInvitationInvalid
	}
	now := common.GetTimestamp()
	var invitation Invitation
	err := pendingInvitations(DB, now).Where("token_hash = ?", hashInvitationToken(token)).First(&invitation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvitationInvalid
	}
	if err != nil {
		return nil, err
	}
	invitation.fillStatus(now)
	return &invitation, nil
}

// SpendInvitationWithTx marks the pending invitation the token opens as used
// by userId. Exactly one row must change, so one token creates one account.
func SpendInvitationWithTx(tx *gorm.DB, token string, userId int) error {
	now := common.GetTimestamp()
	result := pendingInvitations(tx.Model(&Invitation{}), now).
		Where("token_hash = ?", hashInvitationToken(token)).
		Updates(map[string]any{"used_time": now, "used_user_id": userId})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrInvitationInvalid
	}
	return nil
}

// ListInvitations returns invitations newest first, optionally only those of
// one status.
func ListInvitations(status string, startIdx int, num int) ([]*Invitation, int64, error) {
	now := common.GetTimestamp()
	query := DB.Model(&Invitation{})
	switch status {
	case InvitationStatusPending:
		query = pendingInvitations(query, now)
	case InvitationStatusUsed:
		query = query.Where("used_time > 0")
	case InvitationStatusRevoked:
		query = query.Where("used_time = 0 AND revoked_time > 0")
	case InvitationStatusExpired:
		query = query.Where("used_time = 0 AND revoked_time = 0 AND expired_time <= ?", now)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var invitations []*Invitation
	if err := query.Order("id desc").Limit(num).Offset(startIdx).Find(&invitations).Error; err != nil {
		return nil, 0, err
	}
	for _, invitation := range invitations {
		invitation.fillStatus(now)
	}
	return invitations, total, nil
}

// RenewInvitation gives an invitation that is neither used nor revoked a new
// token and a new expiry, so the link mailed before stops working.
func RenewInvitation(id int, validFor int64) (*Invitation, string, error) {
	token, err := newInvitationToken()
	if err != nil {
		return nil, "", err
	}
	now := common.GetTimestamp()
	result := DB.Model(&Invitation{}).Where("id = ? AND used_time = 0 AND revoked_time = 0", id).
		Updates(map[string]any{"token_hash": hashInvitationToken(token), "expired_time": now + validFor})
	if result.Error != nil {
		return nil, "", result.Error
	}
	invitation, err := getInvitationById(id, now)
	if err != nil {
		return nil, "", err
	}
	if result.RowsAffected != 1 {
		return nil, "", ErrInvitationClosed
	}
	return invitation, token, nil
}

// RevokeInvitation stops an invitation that is neither used nor revoked from
// opening the sign-up. The row stays.
func RevokeInvitation(id int) (*Invitation, error) {
	now := common.GetTimestamp()
	result := DB.Model(&Invitation{}).Where("id = ? AND used_time = 0 AND revoked_time = 0", id).Update("revoked_time", now)
	if result.Error != nil {
		return nil, result.Error
	}
	invitation, err := getInvitationById(id, now)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected != 1 {
		return nil, ErrInvitationClosed
	}
	return invitation, nil
}

func getInvitationById(id int, now int64) (*Invitation, error) {
	var invitation Invitation
	err := DB.Where("id = ?", id).First(&invitation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, err
	}
	invitation.fillStatus(now)
	return &invitation, nil
}
