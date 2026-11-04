package controller

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	invitationDisabledMessage = "Invitation registration has been disabled by administrator"
	invitationInvalidMessage  = "This invitation is invalid or has expired"
	invitationLanguageMaxLen  = 16
)

type invitationCreateRequest struct {
	Email    string `json:"email"`
	Language string `json:"language"`
}

// invitationRegisterRequest is the sign-up body. InviteToken is read from the
// request only; it is never stored.
type invitationRegisterRequest struct {
	model.User
	InviteToken string `json:"invite_token"`
}

func CreateInvitation(c *gin.Context) {
	if !common.InvitationRegisterEnabled {
		common.ApiErrorT(c, invitationDisabledMessage)
		return
	}
	var request invitationCreateRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	email, err := service.ValidateAccountEmail(request.Email)
	if err != nil {
		writeSecurityOperationError(c, err)
		return
	}
	language := strings.TrimSpace(request.Language)
	if len(language) > invitationLanguageMaxLen {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if model.IsEmailAlreadyTaken(email) {
		common.ApiErrorT(c, "Email address is already in use")
		return
	}
	hours := common.InvitationValidHours
	invitation, token, err := model.CreateInvitation(email, language, c.GetInt("id"), int64(hours)*3600)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	sendInvitation(c, "invitation.create", invitation, token, hours)
}

func GetInvitations(c *gin.Context) {
	status := c.Query("status")
	if status != "" && !slices.Contains(model.InvitationStatuses, status) {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	pageInfo := common.GetPageQuery(c)
	invitations, total, err := model.ListInvitations(status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(invitations)
	common.ApiSuccess(c, pageInfo)
}

func ResendInvitation(c *gin.Context) {
	if !common.InvitationRegisterEnabled {
		common.ApiErrorT(c, invitationDisabledMessage)
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	hours := common.InvitationValidHours
	invitation, token, err := model.RenewInvitation(id, int64(hours)*3600)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	sendInvitation(c, "invitation.resend", invitation, token, hours)
}

func RevokeInvitation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	invitation, err := model.RevokeInvitation(id)
	if err != nil {
		writeInvitationError(c, err)
		return
	}
	// No account exists for the invited address, so the entry has no target user.
	recordManageAuditFor(c, 0, "invitation.revoke", map[string]any{"invitation_id": invitation.Id, "email": invitation.Email})
	common.ApiSuccess(c, invitation)
}

func CheckInvitation(c *gin.Context) {
	invitation := openInvitation(c, c.Query("token"))
	if invitation == nil {
		return
	}
	common.ApiSuccess(c, gin.H{"email": invitation.Email, "expired_time": invitation.ExpiredTime})
}

// openInvitation returns the pending invitation a sign-up token opens, or
// writes the refusal and returns nil. Every token that opens none gets the
// same refusal.
func openInvitation(c *gin.Context, token string) *model.Invitation {
	if !common.InvitationRegisterEnabled {
		common.ApiErrorT(c, invitationDisabledMessage)
		return nil
	}
	invitation, err := model.GetPendingInvitationByToken(token)
	if err != nil {
		if !errors.Is(err, model.ErrInvitationInvalid) {
			common.SysError(common.LogText("failed to look up an invitation: %s", err.Error()))
		}
		common.ApiErrorT(c, invitationInvalidMessage)
		return nil
	}
	return invitation
}

// insertInvitedUser creates the account and spends the invitation in one
// transaction, so a token spent, revoked or replaced meanwhile creates no
// account.
func insertInvitedUser(user *model.User, token string, inviterId int) error {
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := user.InsertWithTx(tx, inviterId); err != nil {
			return err
		}
		return model.SpendInvitationWithTx(tx, token, user.Id)
	}); err != nil {
		return err
	}
	user.FinishInsert(inviterId)
	return nil
}

// sendInvitation mails the link and answers with it. The link is shown this
// once; the invitation stays pending when the mail fails, so it can be resent.
func sendInvitation(c *gin.Context, action string, invitation *model.Invitation, token string, hours int) {
	link := fmt.Sprintf("%s/sign-up?invite=%s", system_setting.ServerAddress, token)
	lang := invitation.Language
	if lang == "" {
		lang = i18n.StatedLang(c)
	}
	systemName := map[string]any{"SystemName": common.SystemName}
	linkParams := map[string]any{"Link": link}
	subject := i18n.Translate(lang, i18n.MsgEmailInvitationSubject, systemName)
	content := fmt.Sprintf("<p>%s</p>"+
		"<p>%s</p>"+
		"<p>%s </p>"+
		"<p>%s</p>",
		i18n.Translate(lang, i18n.MsgEmailInvitationGreeting, systemName),
		i18n.Translate(lang, i18n.MsgEmailInvitationLink, linkParams),
		i18n.Translate(lang, i18n.MsgEmailInvitationLinkFallback, linkParams),
		i18n.Translate(lang, i18n.MsgEmailInvitationValidity, map[string]any{"Hours": hours}))
	err := common.SendEmail(subject, invitation.Email, content)
	// No account exists for the invited address, so the entry has no target user.
	recordManageAuditFor(c, 0, action, map[string]any{"invitation_id": invitation.Id, "email": invitation.Email, "mail_sent": err == nil})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"invitation": invitation, "link": link})
}

func writeInvitationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrInvitationPending):
		common.ApiErrorT(c, "An invitation to this email address is still pending. Resend it instead.")
	case errors.Is(err, model.ErrInvitationClosed):
		common.ApiErrorT(c, "This invitation has already been used or revoked")
	case errors.Is(err, model.ErrInvitationNotFound):
		common.ApiErrorT(c, "Invitation not found")
	default:
		common.ApiError(c, err)
	}
}
