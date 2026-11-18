package controller_test

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
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
	"github.com/gin-gonic/gin"
	sqlmysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	phoneTestPassword          = "phone-test-password"
	phoneRequiredMessage       = "Phone number is required"
	phoneLengthMessage         = "The phone number is too short or too long"
	phoneFormatMessage         = "The phone number does not match the accepted format"
	phoneCharactersMessage     = "The phone number may contain only digits, spaces, hyphens, dots, brackets and a leading +"
	phoneTakenMessage          = "Phone number is already in use"
	phoneCountryCodeMessage    = "The country calling code must be one to three digits that do not start with 0"
	phoneTrunkPrefixMessage    = "The trunk prefix must be one to three digits"
	phoneNationalNumberMessage = "The national number pattern is not a valid regular expression"
)

// phoneFormats are made-up and real number formats of several countries,
// given as the options an administrator sets.
var (
	phoneTrunkFormat = map[string]string{"PhoneCountryCode": "44", "PhoneTrunkPrefix": "0", "PhoneNationalNumberPattern": `7\d{9}`}
	phonePlainFormat = map[string]string{"PhoneCountryCode": "47", "PhoneTrunkPrefix": "", "PhoneNationalNumberPattern": `[49]\d{7}`}
	phoneSharedCode  = map[string]string{"PhoneCountryCode": "1", "PhoneTrunkPrefix": "1", "PhoneNationalNumberPattern": `[2-9]\d{2}[2-9]\d{6}`}
	phoneNoFormat    = map[string]string{"PhoneCountryCode": "", "PhoneTrunkPrefix": "", "PhoneNationalNumberPattern": ""}
	phoneDefaults    = map[string]string{"RegisterPhoneRequiredEnabled": "false", "PhoneCountryCode": "", "PhoneTrunkPrefix": "", "PhoneNationalNumberPattern": ""}
)

type phoneResponse struct {
	Success    bool           `json:"success"`
	Message    string         `json:"message"`
	MessageKey string         `json:"message_key"`
	Data       map[string]any `json:"data"`
}

type phoneFixture struct {
	engine     *gin.Engine
	adminToken string
}

// setupPhoneTest starts the API router on a fresh database migrated the way
// the server migrates at startup. TEST_SECURITY_DIALECT selects mysql or
// postgres with TEST_MYSQL_DSN or TEST_POSTGRES_DSN; SQLite is the default.
func setupPhoneTest(t *testing.T) *phoneFixture {
	t.Helper()
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedis, previousSecret, previousMaster := common.RedisEnabled, common.SessionSecret, common.IsMasterNode
	previousSQLitePath, previousOptions := common.SQLitePath, common.OptionMap
	previousCritical, previousGlobal := common.CriticalRateLimitEnable, common.GlobalApiRateLimitEnable
	previousDefaultToken := constant.GenerateDefaultToken
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
		constant.GenerateDefaultToken = previousDefaultToken
	})

	dialect := os.Getenv("TEST_SECURITY_DIALECT")
	if dialect == "" {
		dialect = "sqlite"
	}
	t.Setenv("LOG_SQL_DSN", "")
	switch dialect {
	case "sqlite":
		t.Setenv("SQL_DSN", "")
		common.SQLitePath = t.TempDir() + "/phone.db?_pragma=busy_timeout(30000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	default:
		t.Setenv("SQL_DSN", newPhoneTestDatabase(t, dialect))
	}
	common.IsMasterNode, common.RedisEnabled = true, false
	common.SessionSecret = "phone-test-secret"
	require.NoError(t, model.InitDB())
	require.NoError(t, model.InitLogDB())
	model.InitOptionMap()
	t.Logf("database: %s", dialect)
	// Every test starts and ends with the phone options at their defaults, also
	// when it changed them through the API.
	resetPhoneOptions := func() {
		for key, value := range phoneDefaults {
			_ = model.UpdateOption(key, value)
		}
	}
	resetPhoneOptions()
	t.Cleanup(resetPhoneOptions)

	common.CriticalRateLimitEnable, common.GlobalApiRateLimitEnable = false, false
	constant.GenerateDefaultToken = false
	setPhoneOptions(t, map[string]string{
		"RegisterEnabled": "true", "PasswordRegisterEnabled": "true", "EmailVerificationEnabled": "false",
		"EmailDomainRestrictionEnabled": "false", "EmailAliasRestrictionEnabled": "false", "TurnstileCheckEnabled": "false",
	})

	fixture := &phoneFixture{engine: gin.New()}
	require.NotPanics(t, func() { router.SetApiRouter(fixture.engine) })
	fixture.adminToken, _ = createPhoneTestSession(t, "phone-admin", common.RoleAdminUser)
	return fixture
}

// setPhoneOptions saves options as the settings page does and restores the
// previous values when the test ends.
func setPhoneOptions(t *testing.T, options map[string]string) {
	t.Helper()
	for key, value := range options {
		common.OptionMapRWMutex.RLock()
		previous, existed := common.OptionMap[key]
		common.OptionMapRWMutex.RUnlock()
		require.NoError(t, model.UpdateOption(key, value), key)
		if existed {
			t.Cleanup(func() { _ = model.UpdateOption(key, previous) })
		}
	}
}

func requirePhoneNumbers(t *testing.T, format map[string]string) {
	t.Helper()
	options := map[string]string{"RegisterPhoneRequiredEnabled": "true"}
	for key, value := range format {
		options[key] = value
	}
	setPhoneOptions(t, options)
}

func newPhoneTestDatabase(t *testing.T, dialect string) string {
	t.Helper()
	dsn := os.Getenv("TEST_" + strings.ToUpper(dialect) + "_DSN")
	require.NotEmpty(t, dsn, "TEST_%s_DSN is required", strings.ToUpper(dialect))
	name := fmt.Sprintf("newapi_phone_%d", time.Now().UnixNano())
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

// createPhoneTestSession creates an account without a phone number, as every
// account that exists before the option is turned on, and signs it in.
func createPhoneTestSession(t *testing.T, username string, role int) (string, int) {
	t.Helper()
	password, err := common.Password2Hash(phoneTestPassword)
	require.NoError(t, err)
	user := &model.User{Username: username, Password: password, DisplayName: username, Role: role, Status: common.UserStatusEnabled, Group: "default", AffCode: username, AuthVersion: 1}
	require.NoError(t, model.DB.Create(user).Error)
	require.NoError(t, model.PublishUserAuthCache(user.Id))
	bundle, err := service.CreateLoginSession(user.Id, "password", "127.0.0.1", "phone-test")
	require.NoError(t, err)
	return bundle.AccessToken, user.Id
}

func (fixture *phoneFixture) request(t *testing.T, method, path, bearer string, body any) phoneResponse {
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
	var result phoneResponse
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result), response.Body.String())
	return result
}

func (fixture *phoneFixture) register(t *testing.T, username string, extra map[string]any) phoneResponse {
	t.Helper()
	body := map[string]any{"username": username, "password": phoneTestPassword}
	for key, value := range extra {
		body[key] = value
	}
	return fixture.request(t, http.MethodPost, "/api/user/register", "", body)
}

// storedPhone reads the phone column of an account; ok is false when the
// account has no number.
func storedPhone(t *testing.T, username string) (phone string, ok bool) {
	t.Helper()
	var row struct{ Phone sql.NullString }
	require.NoError(t, model.DB.Unscoped().Table("users").Select("phone").Where("username = ?", username).Take(&row).Error)
	return row.Phone.String, row.Phone.Valid
}

func setStoredPhone(t *testing.T, userId int, phone string) {
	t.Helper()
	require.NoError(t, model.DB.Table("users").Where("id = ?", userId).Update("phone", phone).Error)
}

func countUsersNamed(t *testing.T, username string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, model.DB.Unscoped().Model(&model.User{}).Where("username = ?", username).Count(&count).Error)
	return count
}

func userIdNamed(t *testing.T, username string) int {
	t.Helper()
	var user model.User
	require.NoError(t, model.DB.Where("username = ?", username).First(&user).Error)
	return user.Id
}

func TestPhoneNumberOptionOffKeepsRegistrationUnchanged(t *testing.T) {
	fixture := setupPhoneTest(t)

	result := fixture.register(t, "no-phone", nil)
	require.True(t, result.Success, result.Message)
	_, hasPhone := storedPhone(t, "no-phone")
	assert.False(t, hasPhone)

	// With the option off the field is ignored, whatever it holds.
	for username, phone := range map[string]string{"ignored-phone": "+44 7911 123456", "ignored-letters": "call me"} {
		result = fixture.register(t, username, map[string]any{"phone": phone})
		require.True(t, result.Success, "%s: %s", username, result.Message)
		_, hasPhone = storedPhone(t, username)
		assert.False(t, hasPhone, username)
	}

	status := fixture.request(t, http.MethodGet, "/api/status", "", nil)
	require.True(t, status.Success)
	assert.Equal(t, false, status.Data["register_phone_required"])
}

func TestPhoneNumberStatusCarriesTheRequirementAndTheFormat(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)

	status := fixture.request(t, http.MethodGet, "/api/status", "", nil)
	require.True(t, status.Success)
	assert.Equal(t, true, status.Data["register_phone_required"])
	assert.Equal(t, "44", status.Data["phone_country_code"])
	assert.Equal(t, "0", status.Data["phone_trunk_prefix"])
	assert.NotContains(t, status.Data, "phone")
}

func TestRegisterRefusesEachInvalidPhoneNumberWithItsMessage(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)
	result := fixture.register(t, "first-holder", map[string]any{"phone": "07911 123456"})
	require.True(t, result.Success, result.Message)

	for index, test := range []struct {
		name    string
		extra   map[string]any
		message string
	}{
		{"absent", nil, phoneRequiredMessage},
		{"empty", map[string]any{"phone": ""}, phoneRequiredMessage},
		{"separators only", map[string]any{"phone": " - ( ) "}, phoneRequiredMessage},
		{"too short", map[string]any{"phone": "0791"}, phoneLengthMessage},
		{"too long", map[string]any{"phone": "07911 123456 789012"}, phoneLengthMessage},
		{"pattern refuses", map[string]any{"phone": "020 7946 0000"}, phoneFormatMessage},
		{"another country code", map[string]any{"phone": "+33 6 12 34 56 78"}, phoneFormatMessage},
		{"letters", map[string]any{"phone": "07911 12345a"}, phoneCharactersMessage},
		{"extension", map[string]any{"phone": "07911 123457 ext 1"}, phoneCharactersMessage},
		{"other characters", map[string]any{"phone": "07911#123457"}, phoneCharactersMessage},
		{"plus inside", map[string]any{"phone": "0791+1123457"}, phoneCharactersMessage},
		{"held by another account", map[string]any{"phone": "+44 7911 123456"}, phoneTakenMessage},
	} {
		username := fmt.Sprintf("refused-%d", index)
		result := fixture.register(t, username, test.extra)
		assert.False(t, result.Success, test.name)
		assert.Equal(t, test.message, result.Message, test.name)
		assert.Equal(t, test.message, result.MessageKey, test.name)
		assert.Zero(t, countUsersNamed(t, username), "%s creates no account", test.name)
	}
}

func TestRegisterStoresEverySpellingInOneNormalForm(t *testing.T) {
	fixture := setupPhoneTest(t)
	for _, group := range []struct {
		name      string
		format    map[string]string
		spellings map[string]string // spelling -> normal form
	}{
		{"trunk", phoneTrunkFormat, map[string]string{
			"+44 7911 123401":     "07911123401",
			"0044 7911 123402":    "07911123402",
			"44 7911 123403":      "07911123403",
			"07911 123404":        "07911123404",
			"7911 123405":         "07911123405",
			"+44 (0) 7911 123406": "07911123406",
			"(07911) 123-407":     "07911123407",
			"07911.123.408":       "07911123408",
			"[07911] 123 409":     "07911123409",
			"۰۷۹۱۱ ۱۲۳ ۴۱۰":       "07911123410",
			"‏+۴۴ ۷۹۱۱ ۱۲۳۴۱۱‏":   "07911123411",
			"٠٧٩١١١٢٣٤١٢":         "07911123412",
			"०७९११ १२३४१३":        "07911123413",
			"＋４４ ７９１１ １２３４１４":     "07911123414",
			"07911 123415":        "07911123415",
			"07911–123416":        "07911123416",
		}},
		{"plain", phonePlainFormat, map[string]string{
			"+47 412 34 567": "41234567",
			"0047 41234568":  "41234568",
			"47 412 34 569":  "41234569",
			"912 34 570":     "91234570",
		}},
		{"shared", phoneSharedCode, map[string]string{
			"+1 (415) 555-0101": "14155550101",
			"1 415 555 0102":    "14155550102",
			"(415) 555-0103":    "14155550103",
			"001 415 555 0104":  "14155550104",
		}},
		{"international", phoneNoFormat, map[string]string{
			"+1 415 555 0201":  "+14155550201",
			"0049 30 1234567":  "+49301234567",
			"+683 4002":        "+6834002",
			"+۴۹ ۳۰ ۱۲۳۴ ۵۶۷۸": "+493012345678",
		}},
	} {
		requirePhoneNumbers(t, group.format)
		index := 0
		for spelling, normal := range group.spellings {
			index++
			username := fmt.Sprintf("%s-%d", group.name, index)
			result := fixture.register(t, username, map[string]any{"phone": spelling})
			if !assert.True(t, result.Success, "%s %q: %s", group.name, spelling, result.Message) {
				continue
			}
			phone, ok := storedPhone(t, username)
			assert.True(t, ok, spelling)
			assert.Equal(t, normal, phone, "%s %q", group.name, spelling)
		}
	}

	requirePhoneNumbers(t, phoneNoFormat)
	for spelling, message := range map[string]string{
		"415 555 0202":          phoneFormatMessage,
		"+0 415 555 0203":       phoneFormatMessage,
		"+1 415":                phoneLengthMessage,
		"+1 415 555 0204 12345": phoneLengthMessage,
	} {
		result := fixture.register(t, "intl-refused", map[string]any{"phone": spelling})
		assert.False(t, result.Success, spelling)
		assert.Equal(t, message, result.Message, spelling)
	}
}

func TestRegisterPhoneNumberRefusalLeavesTheEmailCodeUsable(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)
	setPhoneOptions(t, map[string]string{"EmailVerificationEnabled": "true"})
	common.RegisterVerificationCodeWithKey("coded@example.com", "424242", common.EmailVerificationPurpose)

	result := fixture.register(t, "coded", map[string]any{"email": "coded@example.com", "verification_code": "424242", "phone": "0791"})
	require.False(t, result.Success)
	assert.Equal(t, phoneLengthMessage, result.Message)
	assert.Zero(t, countUsersNamed(t, "coded"))

	result = fixture.register(t, "coded", map[string]any{"email": "coded@example.com", "verification_code": "424242", "phone": "07911 123499"})
	require.True(t, result.Success, result.Message)
	phone, ok := storedPhone(t, "coded")
	assert.True(t, ok)
	assert.Equal(t, "07911123499", phone)
}

func TestRegisterConcurrentSameNumberMakesOneAccount(t *testing.T) {
	// With email codes on, each sign-up also locks its email. On MySQL those
	// locks can end the losing transaction with a deadlock instead of the
	// unique index; the loser is still refused and the number keeps one
	// holder, but its message depends on timing there.
	exactMessage := func(emailCodes bool) bool {
		return !emailCodes || os.Getenv("TEST_SECURITY_DIALECT") != "mysql"
	}
	for _, emailCodes := range []bool{false, true} {
		t.Run(fmt.Sprintf("email codes %v", emailCodes), func(t *testing.T) {
			fixture := setupPhoneTest(t)
			requirePhoneNumbers(t, phoneTrunkFormat)
			setPhoneOptions(t, map[string]string{"EmailVerificationEnabled": strconv.FormatBool(emailCodes)})
			spellings := []string{"07911 123456", "+44 7911 123456"}
			results := make([]phoneResponse, len(spellings))
			var wg sync.WaitGroup
			for index := range spellings {
				email := fmt.Sprintf("racer-%d@example.com", index)
				common.RegisterVerificationCodeWithKey(email, "424242", common.EmailVerificationPurpose)
				wg.Go(func() {
					results[index] = fixture.register(t, fmt.Sprintf("racer-%d", index), map[string]any{"phone": spellings[index], "email": email, "verification_code": "424242"})
				})
			}
			wg.Wait()

			successes := 0
			for index, result := range results {
				if result.Success {
					successes++
					continue
				}
				assert.Zero(t, countUsersNamed(t, fmt.Sprintf("racer-%d", index)))
				if exactMessage(emailCodes) {
					assert.Equal(t, phoneTakenMessage, result.Message)
				}
			}
			assert.Equal(t, 1, successes, "%+v", results)
			var holders int64
			require.NoError(t, model.DB.Unscoped().Table("users").Where("phone = ?", "07911123456").Count(&holders).Error)
			assert.EqualValues(t, 1, holders)
		})
	}
}

func TestPhoneNumberAccountWithoutNumberSignsInAndReadsItself(t *testing.T) {
	fixture := setupPhoneTest(t)
	_, id := createPhoneTestSession(t, "before-option", common.RoleCommonUser)
	requirePhoneNumbers(t, phoneTrunkFormat)

	login := fixture.request(t, http.MethodPost, "/api/user/login", "", map[string]any{"username": "before-option", "password": phoneTestPassword})
	require.True(t, login.Success, login.Message)
	token, _ := login.Data["access_token"].(string)
	require.NotEmpty(t, token, "%+v", login.Data)
	user, _ := login.Data["user"].(map[string]any)
	assert.Contains(t, user, "phone")
	assert.Nil(t, user["phone"])

	self := fixture.request(t, http.MethodGet, "/api/user/self", token, nil)
	require.True(t, self.Success, self.Message)
	assert.EqualValues(t, id, self.Data["id"])
	assert.Contains(t, self.Data, "phone")
	assert.Nil(t, self.Data["phone"])

	// An account an administrator gave a number reads it in its own data.
	setStoredPhone(t, id, "07911123456")
	self = fixture.request(t, http.MethodGet, "/api/user/self", token, nil)
	require.True(t, self.Success, self.Message)
	assert.Equal(t, "07911123456", self.Data["phone"])
}

func TestPhoneNumberSelfUpdateCannotChangeIt(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)
	token, id := createPhoneTestSession(t, "owner", common.RoleCommonUser)
	setStoredPhone(t, id, "07911123456")

	for _, phone := range []any{"+44 7911 654321", "", nil} {
		result := fixture.request(t, http.MethodPut, "/api/user/self", token, map[string]any{"username": "owner", "display_name": "Owner", "phone": phone})
		require.True(t, result.Success, result.Message)
		stored, ok := storedPhone(t, "owner")
		assert.True(t, ok, "%v", phone)
		assert.Equal(t, "07911123456", stored, "%v", phone)
	}
	var displayName string
	require.NoError(t, model.DB.Table("users").Where("id = ?", id).Select("display_name").Scan(&displayName).Error)
	assert.Equal(t, "Owner", displayName, "the rest of the update applies")
}

func TestAdminSetsChangesAndClearsThePhoneNumber(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)
	_, id := createPhoneTestSession(t, "managed", common.RoleCommonUser)
	_, otherId := createPhoneTestSession(t, "other", common.RoleCommonUser)
	setStoredPhone(t, otherId, "07911000000")
	update := func(phone any) phoneResponse {
		body := map[string]any{"id": id, "username": "managed", "display_name": "managed", "group": "default"}
		if phone != nil {
			body["phone"] = phone
		}
		return fixture.request(t, http.MethodPut, "/api/user/", fixture.adminToken, body)
	}

	result := update("+44 7911 222333")
	require.True(t, result.Success, result.Message)
	phone, ok := storedPhone(t, "managed")
	assert.True(t, ok)
	assert.Equal(t, "07911222333", phone)

	read := fixture.request(t, http.MethodGet, fmt.Sprintf("/api/user/%d", id), fixture.adminToken, nil)
	require.True(t, read.Success, read.Message)
	assert.Equal(t, "07911222333", read.Data["phone"])
	search := fixture.request(t, http.MethodGet, "/api/user/search?keyword=7911222333", fixture.adminToken, nil)
	require.True(t, search.Success, search.Message)
	items, _ := search.Data["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "managed", items[0].(map[string]any)["username"])

	result = update(nil)
	require.True(t, result.Success, result.Message)
	phone, _ = storedPhone(t, "managed")
	assert.Equal(t, "07911222333", phone, "an update without the field keeps the number")

	result = update("0044 7911 000000")
	assert.False(t, result.Success)
	assert.Equal(t, phoneTakenMessage, result.Message)
	result = update("020 7946 0000")
	assert.False(t, result.Success)
	assert.Equal(t, phoneFormatMessage, result.Message)
	phone, _ = storedPhone(t, "managed")
	assert.Equal(t, "07911222333", phone, "a refused update keeps the number")

	result = update("")
	require.True(t, result.Success, result.Message)
	_, ok = storedPhone(t, "managed")
	assert.False(t, ok, "an empty number clears it")
	// Two accounts without a number do not collide.
	_, ok = storedPhone(t, "phone-admin")
	assert.False(t, ok)
}

func TestAdminCreateUserMayTakeAPhoneNumber(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)
	create := func(username string, phone any) phoneResponse {
		body := map[string]any{"username": username, "password": phoneTestPassword, "display_name": username, "role": common.RoleCommonUser}
		if phone != nil {
			body["phone"] = phone
		}
		return fixture.request(t, http.MethodPost, "/api/user/", fixture.adminToken, body)
	}

	result := create("created-without", nil)
	require.True(t, result.Success, result.Message)
	_, ok := storedPhone(t, "created-without")
	assert.False(t, ok)

	result = create("created-with", "7911 444555")
	require.True(t, result.Success, result.Message)
	phone, ok := storedPhone(t, "created-with")
	assert.True(t, ok)
	assert.Equal(t, "07911444555", phone)

	result = create("created-taken", "+44 7911 444555")
	assert.False(t, result.Success)
	assert.Equal(t, phoneTakenMessage, result.Message)
	result = create("created-letters", "07911 44455x")
	assert.False(t, result.Success)
	assert.Equal(t, phoneCharactersMessage, result.Message)
	assert.Zero(t, countUsersNamed(t, "created-taken")+countUsersNamed(t, "created-letters"))
}

func TestPhoneNumberOfADeletedAccountIsFreeOnlyWhenTheAccountIsGone(t *testing.T) {
	fixture := setupPhoneTest(t)
	requirePhoneNumbers(t, phoneTrunkFormat)
	result := fixture.register(t, "leaving", map[string]any{"phone": "07911 555666"})
	require.True(t, result.Success, result.Message)
	id := userIdNamed(t, "leaving")

	_, err := model.DeleteUserById(id)
	require.NoError(t, err)
	result = fixture.register(t, "after-soft-delete", map[string]any{"phone": "07911 555666"})
	assert.False(t, result.Success, "a soft-deleted account keeps its number")
	assert.Equal(t, phoneTakenMessage, result.Message)

	_, err = model.HardDeleteUserById(id)
	require.NoError(t, err)
	result = fixture.register(t, "after-hard-delete", map[string]any{"phone": "07911 555666"})
	require.True(t, result.Success, result.Message)
	phone, _ := storedPhone(t, "after-hard-delete")
	assert.Equal(t, "07911555666", phone)
}

func TestPhoneNumberFormatOptionsAreValidated(t *testing.T) {
	fixture := setupPhoneTest(t)
	rootToken, _ := createPhoneTestSession(t, "phone-root", common.RoleRootUser)
	for _, test := range []struct {
		key, value, message string
	}{
		{"PhoneCountryCode", "0", phoneCountryCodeMessage},
		{"PhoneCountryCode", "1234", phoneCountryCodeMessage},
		{"PhoneCountryCode", "+44", phoneCountryCodeMessage},
		{"PhoneCountryCode", "44", ""},
		{"PhoneCountryCode", "", ""},
		{"PhoneTrunkPrefix", "0a", phoneTrunkPrefixMessage},
		{"PhoneTrunkPrefix", "0000", phoneTrunkPrefixMessage},
		{"PhoneTrunkPrefix", "0", ""},
		{"PhoneNationalNumberPattern", `7[0-9`, phoneNationalNumberMessage},
		{"PhoneNationalNumberPattern", `(`, phoneNationalNumberMessage},
		{"PhoneNationalNumberPattern", `7\d{9}`, ""},
		{"RegisterPhoneRequiredEnabled", "true", ""},
	} {
		result := fixture.request(t, http.MethodPut, "/api/option/", rootToken, map[string]any{"key": test.key, "value": test.value})
		if test.message == "" {
			assert.True(t, result.Success, "%s=%q: %s", test.key, test.value, result.Message)
			continue
		}
		assert.False(t, result.Success, "%s=%q", test.key, test.value)
		assert.Equal(t, test.message, result.Message, "%s=%q", test.key, test.value)
	}
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	assert.Equal(t, `7\d{9}`, common.OptionMap["PhoneNationalNumberPattern"])
	assert.Equal(t, "0", common.OptionMap["PhoneTrunkPrefix"])
}
