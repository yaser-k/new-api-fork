package controller_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/router"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	sqlmysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	invitationTestServer        = "https://console.example.com"
	invitationDisabledMessage   = "Invitation registration has been disabled by administrator"
	invitationInvalidMessage    = "This invitation is invalid or has expired"
	invitationTestPassword      = "invited-password"
	invitationTestSystemName    = "New API"
	invitationTestDefaultEmail  = "invitee@example.com"
	invitationTestPendingFilter = "pending"
)

var invitationLinkPattern = regexp.MustCompile(`^` + regexp.QuoteMeta(invitationTestServer) + `/sign-up\?invite=([A-Za-z0-9_-]{43})$`)

type invitationResponse struct {
	Success    bool           `json:"success"`
	Message    string         `json:"message"`
	MessageKey string         `json:"message_key"`
	Data       map[string]any `json:"data"`
}

type invitationMail struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type invitationFixture struct {
	engine      *gin.Engine
	adminToken  string
	userToken   string
	rootToken   string
	adminId     int
	mu          sync.Mutex
	mails       []invitationMail
	failSending bool
}

// setupInvitationTest starts the API router on a fresh database migrated the
// way the server migrates at startup. TEST_SECURITY_DIALECT selects mysql or
// postgres with TEST_MYSQL_DSN or TEST_POSTGRES_DSN; SQLite is the default.
func setupInvitationTest(t *testing.T) *invitationFixture {
	t.Helper()
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedis, previousSecret, previousMaster := common.RedisEnabled, common.SessionSecret, common.IsMasterNode
	previousSQLitePath, previousOptions := common.SQLitePath, common.OptionMap
	previousCritical, previousGlobal := common.CriticalRateLimitEnable, common.GlobalApiRateLimitEnable
	previousServer, previousName := system_setting.ServerAddress, common.SystemName
	previousDefaultToken, previousTransport := constant.GenerateDefaultToken, http.DefaultTransport
	t.Cleanup(func() {
		if model.DB != nil && model.DB != previousDB {
			if connection, err := model.DB.DB(); err == nil {
				_ = connection.Close()
			}
		}
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMain, previousLog)
		common.RedisEnabled, common.SessionSecret, common.IsMasterNode = previousRedis, previousSecret, previousMaster
		common.SQLitePath = previousSQLitePath
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
		common.CriticalRateLimitEnable, common.GlobalApiRateLimitEnable = previousCritical, previousGlobal
		system_setting.ServerAddress, common.SystemName = previousServer, previousName
		constant.GenerateDefaultToken, http.DefaultTransport = previousDefaultToken, previousTransport
	})

	dialect := os.Getenv("TEST_SECURITY_DIALECT")
	if dialect == "" {
		dialect = "sqlite"
	}
	t.Setenv("LOG_SQL_DSN", "")
	switch dialect {
	case "sqlite":
		t.Setenv("SQL_DSN", "")
		common.SQLitePath = t.TempDir() + "/invitation.db?_pragma=busy_timeout(30000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	default:
		t.Setenv("SQL_DSN", newInvitationTestDatabase(t, dialect))
	}
	common.IsMasterNode, common.RedisEnabled = true, false
	common.SessionSecret = "invitation-test-secret"
	require.NoError(t, model.InitDB())
	require.NoError(t, model.InitLogDB())
	model.InitOptionMap()
	t.Logf("database: %s", dialect)

	common.CriticalRateLimitEnable, common.GlobalApiRateLimitEnable = false, false
	system_setting.ServerAddress, common.SystemName = invitationTestServer, invitationTestSystemName
	constant.GenerateDefaultToken = false
	fixture := &invitationFixture{}
	http.DefaultTransport = invitationMailTransport(func(request *http.Request) (*http.Response, error) {
		var mail invitationMail
		if err := common.DecodeJson(request.Body, &mail); err != nil {
			return nil, err
		}
		fixture.mu.Lock()
		defer fixture.mu.Unlock()
		if fixture.failSending {
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
		}
		fixture.mails = append(fixture.mails, mail)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"invitation-mail"}`)), Header: make(http.Header)}, nil
	})
	setInvitationOptions(t, map[string]string{
		"ResendApiKey": "test-key", "ResendFrom": "sender@example.com", "EmailProvider": "resend",
		"RegisterEnabled": "true", "PasswordRegisterEnabled": "true", "EmailVerificationEnabled": "false",
		"EmailDomainRestrictionEnabled": "false", "EmailAliasRestrictionEnabled": "false", "TurnstileCheckEnabled": "false",
		"InvitationRegisterEnabled": "true", "InvitationValidHours": "72",
	})

	fixture.engine = gin.New()
	require.NotPanics(t, func() { router.SetApiRouter(fixture.engine) })
	var adminId int
	fixture.adminToken, adminId = createInvitationTestSession(t, "invitation-admin", common.RoleAdminUser)
	fixture.adminId = adminId
	fixture.userToken, _ = createInvitationTestSession(t, "invitation-user", common.RoleCommonUser)
	fixture.rootToken, _ = createInvitationTestSession(t, "invitation-root", common.RoleRootUser)
	return fixture
}

type invitationMailTransport func(*http.Request) (*http.Response, error)

func (transport invitationMailTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

// setInvitationOptions saves options as the settings page does and restores
// the previous values when the test ends.
func setInvitationOptions(t *testing.T, options map[string]string) {
	t.Helper()
	// Resend's key and sender must be stored before the provider is selected.
	for _, key := range []string{"ResendApiKey", "ResendFrom", "EmailProvider"} {
		if value, ok := options[key]; ok {
			setInvitationOption(t, key, value)
		}
	}
	for key, value := range options {
		if key != "ResendApiKey" && key != "ResendFrom" && key != "EmailProvider" {
			setInvitationOption(t, key, value)
		}
	}
}

func setInvitationOption(t *testing.T, key, value string) {
	t.Helper()
	common.OptionMapRWMutex.RLock()
	previous, existed := common.OptionMap[key]
	common.OptionMapRWMutex.RUnlock()
	require.NoError(t, model.UpdateOption(key, value), key)
	if existed {
		t.Cleanup(func() { _ = model.UpdateOption(key, previous) })
	}
}

func newInvitationTestDatabase(t *testing.T, dialect string) string {
	t.Helper()
	dsn := os.Getenv("TEST_" + strings.ToUpper(dialect) + "_DSN")
	require.NotEmpty(t, dsn, "TEST_%s_DSN is required", strings.ToUpper(dialect))
	name := fmt.Sprintf("newapi_invitation_%d", time.Now().UnixNano())
	var admin gorm.Dialector
	var isolated, create string
	switch dialect {
	case "mysql":
		config, err := sqlmysql.ParseDSN(dsn)
		require.NoError(t, err)
		admin = mysql.Open(dsn)
		config.DBName = name
		isolated = config.FormatDSN()
		create = "CREATE DATABASE " + name + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
	case "postgres":
		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		admin = postgres.Open(dsn)
		parsed.Path = "/" + name
		isolated = parsed.String()
		create = "CREATE DATABASE " + name
	default:
		t.Fatalf("unsupported dialect %s", dialect)
	}
	db, err := gorm.Open(admin, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(create).Error)
	t.Cleanup(func() {
		if model.DB != nil {
			if connection, err := model.DB.DB(); err == nil {
				_ = connection.Close()
			}
		}
		_ = db.Exec("DROP DATABASE " + name).Error
		if connection, err := db.DB(); err == nil {
			_ = connection.Close()
		}
	})
	return isolated
}

func createInvitationTestSession(t *testing.T, username string, role int) (string, int) {
	t.Helper()
	password, err := common.Password2Hash("invitation-test-password")
	require.NoError(t, err)
	user := &model.User{Username: username, Password: password, Role: role, Status: common.UserStatusEnabled, Group: "default", AffCode: username, AuthVersion: 1}
	require.NoError(t, model.DB.Create(user).Error)
	require.NoError(t, model.PublishUserAuthCache(user.Id))
	bundle, err := service.CreateLoginSession(user.Id, "password", "127.0.0.1", "invitation-test")
	require.NoError(t, err)
	return bundle.AccessToken, user.Id
}

func (fixture *invitationFixture) request(t *testing.T, method, path, bearer string, body any) (int, invitationResponse, string) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := common.Marshal(body)
		require.NoError(t, err)
		reader = strings.NewReader(string(encoded))
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	response := httptest.NewRecorder()
	fixture.engine.ServeHTTP(response, request)
	var result invitationResponse
	_ = common.Unmarshal(response.Body.Bytes(), &result)
	return response.Code, result, response.Body.String()
}

// invite creates an invitation as the administrator and returns its id and
// the token of the link the response carries.
func (fixture *invitationFixture) invite(t *testing.T, email, language string) (int, string) {
	t.Helper()
	status, result, raw := fixture.request(t, http.MethodPost, "/api/user/invitation", fixture.adminToken, map[string]string{"email": email, "language": language})
	require.Equal(t, http.StatusOK, status, raw)
	require.True(t, result.Success, raw)
	return invitationIdAndToken(t, result)
}

func invitationIdAndToken(t *testing.T, result invitationResponse) (int, string) {
	t.Helper()
	invitation, ok := result.Data["invitation"].(map[string]any)
	require.True(t, ok, "the response carries the invitation: %v", result.Data)
	link, _ := result.Data["link"].(string)
	match := invitationLinkPattern.FindStringSubmatch(link)
	require.Len(t, match, 2, "link %q", link)
	id, ok := invitation["id"].(float64)
	require.True(t, ok)
	return int(id), match[1]
}

func (fixture *invitationFixture) register(t *testing.T, body map[string]any) invitationResponse {
	t.Helper()
	_, result, _ := fixture.request(t, http.MethodPost, "/api/user/register", "", body)
	return result
}

func (fixture *invitationFixture) check(t *testing.T, token string) invitationResponse {
	t.Helper()
	_, result, _ := fixture.request(t, http.MethodGet, "/api/invitation/check?token="+url.QueryEscape(token), "", nil)
	return result
}

func (fixture *invitationFixture) sentMails() []invitationMail {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return append([]invitationMail(nil), fixture.mails...)
}

func countUsersWithEmail(t *testing.T, email string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, model.DB.Model(&model.User{}).Where("email = ?", email).Count(&count).Error)
	return count
}

type invitationRecord struct {
	Email       string
	TokenHash   string
	CreatedBy   int
	CreatedTime int64
	ExpiredTime int64
	UsedTime    int64
	UsedUserId  int
	RevokedTime int64
}

func invitationRow(t *testing.T, id int) invitationRecord {
	t.Helper()
	var record invitationRecord
	require.NoError(t, model.DB.Table("invitations").Where("id = ?", id).Take(&record).Error)
	return record
}

func invitationTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TestInvitationRoutesRegisterBesideExistingRoutes(t *testing.T) {
	fixture := setupInvitationTest(t)
	registered := map[string]bool{}
	for _, route := range fixture.engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"POST /api/user/invitation",
		"GET /api/user/invitation",
		"POST /api/user/invitation/:id/resend",
		"POST /api/user/invitation/:id/revoke",
		"GET /api/invitation/check",
		"POST /api/user/",
		"GET /api/user/:id",
		"GET /api/verification",
	} {
		assert.True(t, registered[route], "route %s is registered", route)
	}

	_, result, _ := fixture.request(t, http.MethodGet, "/api/status", "", nil)
	require.True(t, result.Success)
	assert.Equal(t, true, result.Data["invitation_register_enabled"])
	setInvitationOptions(t, map[string]string{"InvitationRegisterEnabled": "false"})
	_, result, _ = fixture.request(t, http.MethodGet, "/api/status", "", nil)
	assert.Equal(t, false, result.Data["invitation_register_enabled"])
}

func TestInvitationValidHoursOptionRange(t *testing.T) {
	fixture := setupInvitationTest(t)
	for _, test := range []struct {
		value    string
		accepted bool
	}{{"0", false}, {"721", false}, {"-1", false}, {"abc", false}, {"1", true}, {"720", true}} {
		_, result, raw := fixture.request(t, http.MethodPut, "/api/option/", fixture.rootToken, map[string]any{"key": "InvitationValidHours", "value": test.value})
		assert.Equal(t, test.accepted, result.Success, "InvitationValidHours=%s: %s", test.value, raw)
	}
	common.OptionMapRWMutex.RLock()
	stored := common.OptionMap["InvitationValidHours"]
	common.OptionMapRWMutex.RUnlock()
	assert.Equal(t, "720", stored)
}

func TestRegisterWithoutInvitationKeepsClosedRegistrationRefusals(t *testing.T) {
	fixture := setupInvitationTest(t)
	body := map[string]any{"username": "closed-user", "password": invitationTestPassword, "email": "closed@example.com"}

	setInvitationOptions(t, map[string]string{"RegisterEnabled": "false"})
	result := fixture.register(t, body)
	assert.False(t, result.Success)
	assert.Equal(t, "New user registration has been disabled by administrator", result.Message)

	setInvitationOptions(t, map[string]string{"RegisterEnabled": "true", "PasswordRegisterEnabled": "false"})
	result = fixture.register(t, body)
	assert.False(t, result.Success)
	assert.Equal(t, "Password registration has been disabled by administrator, please use third-party account verification", result.Message)

	var count int64
	require.NoError(t, model.DB.Model(&model.User{}).Where("username = ?", "closed-user").Count(&count).Error)
	assert.Zero(t, count)
}

func TestInvitationOptionOffRefusesEveryInvitationCall(t *testing.T) {
	fixture := setupInvitationTest(t)
	id, token := fixture.invite(t, invitationTestDefaultEmail, "en")
	setInvitationOptions(t, map[string]string{"InvitationRegisterEnabled": "false"})

	results := map[string]invitationResponse{}
	_, results["create"], _ = fixture.request(t, http.MethodPost, "/api/user/invitation", fixture.adminToken, map[string]string{"email": "second@example.com", "language": "en"})
	_, results["resend"], _ = fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/resend", id), fixture.adminToken, nil)
	results["check"] = fixture.check(t, token)
	results["register"] = fixture.register(t, map[string]any{"username": "invited", "password": invitationTestPassword, "invite_token": token})
	for name, result := range results {
		assert.False(t, result.Success, name)
		assert.Equal(t, invitationDisabledMessage, result.Message, name)
	}
	assert.Zero(t, countUsersWithEmail(t, invitationTestDefaultEmail))
	assert.Zero(t, countUsersWithEmail(t, "second@example.com"))
}

func TestInvitationRegisterCreatesOneAccountWithTheInvitedEmail(t *testing.T) {
	fixture := setupInvitationTest(t)
	id, token := fixture.invite(t, invitationTestDefaultEmail, "en")
	// The invitation is the permission: closed registration and a required
	// email code do not apply to it.
	setInvitationOptions(t, map[string]string{"RegisterEnabled": "false", "PasswordRegisterEnabled": "false", "EmailVerificationEnabled": "true"})
	constant.GenerateDefaultToken = true

	result := fixture.register(t, map[string]any{"username": "invited", "password": invitationTestPassword, "email": "attacker@example.com", "invite_token": token})
	require.True(t, result.Success, result.Message)
	var user model.User
	require.NoError(t, model.DB.Where("username = ?", "invited").First(&user).Error)
	assert.Equal(t, invitationTestDefaultEmail, user.Email)
	assert.Equal(t, common.RoleCommonUser, user.Role)
	assert.Zero(t, countUsersWithEmail(t, "attacker@example.com"))
	var tokens int64
	require.NoError(t, model.DB.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&tokens).Error)
	assert.EqualValues(t, 1, tokens, "the default token is created as for any sign-up")

	row := invitationRow(t, id)
	assert.NotZero(t, row.UsedTime)
	assert.Equal(t, user.Id, row.UsedUserId)

	result = fixture.register(t, map[string]any{"username": "invited-again", "password": invitationTestPassword, "invite_token": token})
	assert.False(t, result.Success)
	assert.Equal(t, invitationInvalidMessage, result.Message)
	assert.EqualValues(t, 1, countUsersWithEmail(t, invitationTestDefaultEmail))
	assert.Equal(t, invitationInvalidMessage, fixture.check(t, token).Message)
}

func TestInvitationRefusesExpiredRevokedAndUnknownTokensAlike(t *testing.T) {
	fixture := setupInvitationTest(t)
	expiredId, expired := fixture.invite(t, "expired@example.com", "en")
	require.NoError(t, model.DB.Table("invitations").Where("id = ?", expiredId).Update("expired_time", time.Now().Add(-time.Minute).Unix()).Error)
	revokedId, revoked := fixture.invite(t, "revoked@example.com", "en")
	_, result, raw := fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/revoke", revokedId), fixture.adminToken, nil)
	require.True(t, result.Success, raw)
	_, valid := fixture.invite(t, "valid@example.com", "en")
	unknown := strings.Repeat("A", 43)

	check := fixture.check(t, valid)
	require.True(t, check.Success, check.Message)
	assert.Equal(t, "valid@example.com", check.Data["email"])
	assert.NotZero(t, check.Data["expired_time"])

	for name, token := range map[string]string{"expired": expired, "revoked": revoked, "unknown": unknown, "empty": ""} {
		check := fixture.check(t, token)
		assert.False(t, check.Success, name)
		assert.Equal(t, invitationInvalidMessage, check.Message, name)
		assert.Nil(t, check.Data, name)
		if token == "" {
			continue
		}
		registered := fixture.register(t, map[string]any{"username": "user-" + name, "password": invitationTestPassword, "invite_token": token})
		assert.False(t, registered.Success, name)
		assert.Equal(t, invitationInvalidMessage, registered.Message, name)
	}
	assert.Zero(t, countUsersWithEmail(t, "expired@example.com"))
	assert.Zero(t, countUsersWithEmail(t, "revoked@example.com"))
}

func TestInvitationConcurrentRegistersMakeOneAccount(t *testing.T) {
	fixture := setupInvitationTest(t)
	_, token := fixture.invite(t, invitationTestDefaultEmail, "en")
	var wg sync.WaitGroup
	results := make([]invitationResponse, 2)
	for index := range results {
		wg.Go(func() {
			results[index] = fixture.register(t, map[string]any{"username": fmt.Sprintf("racer-%d", index), "password": invitationTestPassword, "invite_token": token})
		})
	}
	wg.Wait()
	successes := 0
	for _, result := range results {
		if result.Success {
			successes++
		}
	}
	assert.Equal(t, 1, successes, "%+v", results)
	assert.EqualValues(t, 1, countUsersWithEmail(t, invitationTestDefaultEmail))
}

func TestAdminCreateInvitation(t *testing.T) {
	fixture := setupInvitationTest(t)
	require.NoError(t, model.DB.Create(&model.User{Username: "holder", Password: "unused", Email: "taken@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "holder"}).Error)
	setInvitationOptions(t, map[string]string{"InvitationValidHours": "5"})

	id, token := fixture.invite(t, " Invitee@Example.com ", "en")
	row := invitationRow(t, id)
	assert.Equal(t, invitationTestDefaultEmail, row.Email)
	assert.Equal(t, invitationTokenHash(token), row.TokenHash)
	assert.Equal(t, fixture.adminId, row.CreatedBy)
	assert.EqualValues(t, 5*3600, row.ExpiredTime-row.CreatedTime)
	columns := map[string]any{}
	require.NoError(t, model.DB.Table("invitations").Where("id = ?", id).Take(&columns).Error)
	for column, value := range columns {
		if raw, ok := value.([]byte); ok {
			value = string(raw)
		}
		assert.NotContains(t, fmt.Sprint(value), token, "column %s holds the raw token", column)
	}

	mails := fixture.sentMails()
	require.Len(t, mails, 1)
	assert.Equal(t, []string{invitationTestDefaultEmail}, mails[0].To)
	assert.Contains(t, mails[0].HTML, invitationTestServer+"/sign-up?invite="+token)
	assert.Contains(t, mails[0].HTML, "5 hours")

	for name, email := range map[string]string{"taken": "taken@example.com", "pending": "INVITEE@example.com", "invalid": "not-an-address"} {
		_, result, _ := fixture.request(t, http.MethodPost, "/api/user/invitation", fixture.adminToken, map[string]string{"email": email, "language": "en"})
		assert.False(t, result.Success, name)
	}
	setInvitationOptions(t, map[string]string{"EmailDomainWhitelist": "allowed.example", "EmailDomainRestrictionEnabled": "true"})
	_, result, _ := fixture.request(t, http.MethodPost, "/api/user/invitation", fixture.adminToken, map[string]string{"email": "outsider@example.com", "language": "en"})
	assert.False(t, result.Success)
	assert.Equal(t, "This email address is not allowed by the administrator's email policy.", result.Message)
	assert.Len(t, fixture.sentMails(), 1)

	_, list, raw := fixture.request(t, http.MethodGet, "/api/user/invitation?p=1&page_size=10&status="+invitationTestPendingFilter, fixture.adminToken, nil)
	require.True(t, list.Success, raw)
	assert.EqualValues(t, 1, list.Data["total"])
	items, _ := list.Data["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, invitationTestDefaultEmail, item["email"])
	assert.Equal(t, "pending", item["status"])
	assert.NotContains(t, raw, token)
	assert.NotContains(t, raw, invitationTokenHash(token))
	assert.NotContains(t, raw, "token_hash")
	_, list, _ = fixture.request(t, http.MethodGet, "/api/user/invitation?status=used", fixture.adminToken, nil)
	assert.EqualValues(t, 0, list.Data["total"])

	var events []model.AuditLog
	require.NoError(t, model.LOG_DB.Where("action = ?", "invitation.create").Find(&events).Error)
	require.Len(t, events, 1)
	encoded, err := common.Marshal(events[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), invitationTestDefaultEmail)
	assert.Contains(t, string(encoded), `"invitation_id":`)
	assert.NotContains(t, string(encoded), token)
}

func TestAdminCreateInvitationKeepsTheRowWhenTheMailFails(t *testing.T) {
	fixture := setupInvitationTest(t)
	fixture.failSending = true
	_, result, _ := fixture.request(t, http.MethodPost, "/api/user/invitation", fixture.adminToken, map[string]string{"email": invitationTestDefaultEmail, "language": "en"})
	assert.False(t, result.Success)
	assert.Equal(t, "Resend email delivery failed (HTTP 500)", result.Message)

	_, list, raw := fixture.request(t, http.MethodGet, "/api/user/invitation", fixture.adminToken, nil)
	require.True(t, list.Success, raw)
	items, _ := list.Data["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "pending", item["status"])

	fixture.failSending = false
	_, resent, raw := fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%v/resend", item["id"]), fixture.adminToken, nil)
	require.True(t, resent.Success, raw)
	_, token := invitationIdAndToken(t, resent)
	assert.True(t, fixture.check(t, token).Success)
}

func TestAdminInvitationRoutesRefuseOrdinaryAndAnonymousCallers(t *testing.T) {
	fixture := setupInvitationTest(t)
	id, _ := fixture.invite(t, invitationTestDefaultEmail, "en")
	for _, caller := range []struct{ name, token string }{{"anonymous", ""}, {"ordinary user", fixture.userToken}} {
		// The admin group's own refusal, as an existing admin route gives it.
		wantStatus, _, wantBody := fixture.request(t, http.MethodGet, "/api/user/search?keyword=x", caller.token, nil)
		require.NotEqual(t, http.StatusOK, wantStatus)
		for _, route := range []struct{ method, path string }{
			{http.MethodPost, "/api/user/invitation"},
			{http.MethodGet, "/api/user/invitation"},
			{http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/resend", id)},
			{http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/revoke", id)},
		} {
			status, _, body := fixture.request(t, route.method, route.path, caller.token, map[string]string{"email": "other@example.com", "language": "en"})
			assert.Equal(t, wantStatus, status, "%s %s %s", caller.name, route.method, route.path)
			assert.JSONEq(t, wantBody, body, "%s %s %s", caller.name, route.method, route.path)
		}
	}
	assert.Equal(t, "pending", invitationStatus(t, fixture, id))
	assert.Len(t, fixture.sentMails(), 1)
}

func invitationStatus(t *testing.T, fixture *invitationFixture, id int) string {
	t.Helper()
	_, list, raw := fixture.request(t, http.MethodGet, "/api/user/invitation?page_size=100", fixture.adminToken, nil)
	require.True(t, list.Success, raw)
	items, _ := list.Data["items"].([]any)
	for _, item := range items {
		fields := item.(map[string]any)
		if fields["id"] == float64(id) {
			return fields["status"].(string)
		}
	}
	t.Fatalf("invitation %d is not listed: %s", id, raw)
	return ""
}

func TestInvitationResendAndRevoke(t *testing.T) {
	fixture := setupInvitationTest(t)
	id, first := fixture.invite(t, invitationTestDefaultEmail, "en")
	_, resent, raw := fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/resend", id), fixture.adminToken, nil)
	require.True(t, resent.Success, raw)
	_, second := invitationIdAndToken(t, resent)
	assert.NotEqual(t, first, second)
	assert.Equal(t, invitationInvalidMessage, fixture.check(t, first).Message)
	assert.True(t, fixture.check(t, second).Success)
	mails := fixture.sentMails()
	require.Len(t, mails, 2)
	assert.Contains(t, mails[1].HTML, second)
	assert.NotContains(t, mails[1].HTML, first)

	// An expired invitation can be sent again with a new expiry.
	require.NoError(t, model.DB.Table("invitations").Where("id = ?", id).Update("expired_time", time.Now().Add(-time.Minute).Unix()).Error)
	assert.Equal(t, "expired", invitationStatus(t, fixture, id))
	_, resent, raw = fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/resend", id), fixture.adminToken, nil)
	require.True(t, resent.Success, raw)
	_, third := invitationIdAndToken(t, resent)
	assert.Equal(t, "pending", invitationStatus(t, fixture, id))

	_, revoked, raw := fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/revoke", id), fixture.adminToken, nil)
	require.True(t, revoked.Success, raw)
	assert.Equal(t, invitationInvalidMessage, fixture.check(t, third).Message)
	assert.Equal(t, "revoked", invitationStatus(t, fixture, id))
	assert.NotZero(t, invitationRow(t, id).RevokedTime)
	_, again, _ := fixture.request(t, http.MethodPost, fmt.Sprintf("/api/user/invitation/%d/resend", id), fixture.adminToken, nil)
	assert.False(t, again.Success, "a revoked invitation cannot be sent again")
	result := fixture.register(t, map[string]any{"username": "late", "password": invitationTestPassword, "invite_token": third})
	assert.Equal(t, invitationInvalidMessage, result.Message)

	for _, action := range []string{"invitation.create", "invitation.resend", "invitation.revoke"} {
		var events []model.AuditLog
		require.NoError(t, model.LOG_DB.Where("action = ?", action).Find(&events).Error)
		require.NotEmpty(t, events, action)
		for _, event := range events {
			encoded, err := common.Marshal(event)
			require.NoError(t, err)
			assert.Contains(t, string(encoded), invitationTestDefaultEmail, action)
			for _, token := range []string{first, second, third} {
				assert.NotContains(t, string(encoded), token, action)
			}
		}
	}
}

func TestInvitationMailLanguage(t *testing.T) {
	type mailLines struct{ subject, greeting, link, fallback, validity string }
	cases := map[string]func(link string) mailLines{
		"en": func(link string) mailLines {
			return mailLines{
				subject:  invitationTestSystemName + " — Invitation to create an account",
				greeting: "Hello, you have been invited to create an account on " + invitationTestSystemName + ".",
				link:     "Click <a href='" + link + "'>here</a> to create your account.",
				fallback: "If the link does not open, copy the link below into your browser:<br> " + link,
				validity: "This invitation is valid for 72 hours and can be used once. If you were not expecting it, please ignore this email.",
			}
		},
		// U+200C joins شده‌اید and می‌توان; U+2068 and U+2069 isolate the link.
		"fa": func(link string) mailLines {
			return mailLines{
				subject:  "دعوت به ساخت حساب در " + invitationTestSystemName,
				greeting: "سلام، شما برای ساخت حساب در " + invitationTestSystemName + " دعوت شده‌اید.",
				link:     "برای ساخت حساب خود، <a href='" + link + "'>اینجا</a> کلیک کنید.",
				fallback: "اگر پیوند باز نشد، پیوند زیر را در مرورگر خود کپی کنید:<br> ⁨" + link + "⁩",
				validity: "این دعوت تا 72 ساعت معتبر است و فقط یک بار می‌توان از آن استفاده کرد. اگر انتظار آن را نداشتید، این ایمیل را نادیده بگیرید.",
			}
		},
	}
	for language, expected := range cases {
		t.Run(language, func(t *testing.T) {
			if !slices.Contains(i18n.SupportedLanguages(), language) {
				t.Skipf("the backend has no %s locale", language)
			}
			fixture := setupInvitationTest(t)
			_, token := fixture.invite(t, invitationTestDefaultEmail, language)
			mails := fixture.sentMails()
			require.Len(t, mails, 1)
			lines := expected(invitationTestServer + "/sign-up?invite=" + token)
			assert.Equal(t, lines.subject, mails[0].Subject)
			for _, line := range []string{lines.greeting, lines.link, lines.fallback, lines.validity} {
				assert.Contains(t, mails[0].HTML, "<p>"+line, language)
			}
		})
	}
}
