package model

import (
	"context"
	"errors"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/common"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// An account's phone number is stored in one normal form, the full
// international number (+, country code, national number), which does not
// depend on the format options. Two spellings of one number, under any
// options, are one number, and the unique index on users.phone holds one
// account per number. Registration, the administrator's edits and a later
// verification step all read numbers with PhoneNumberFormat.Normalize and check
// them with EnsurePhoneNumberAvailable; PhoneNumberFormat.Display writes the
// stored number nationally, for showing it only.

// The refusals are web console messages, so the API answers them as they are.
var (
	ErrPhoneNumberMissing    = common.NewMessage("Phone number is required")
	ErrPhoneNumberLength     = common.NewMessage("The phone number is too short or too long")
	ErrPhoneNumberFormat     = common.NewMessage("The phone number does not match the accepted format")
	ErrPhoneNumberCharacters = common.NewMessage("The phone number may contain only digits, spaces, hyphens, dots, brackets and a leading +")
	ErrPhoneNumberTaken      = common.NewMessage("Phone number is already in use")
	// ErrPhoneNumberBusy answers a sign-up RetryPhoneNumberRace gave up on.
	ErrPhoneNumberBusy = common.NewMessage("The sign-up could not be completed right now. Please try again.")
)

const (
	// An international number, country code included, has 7 to 15 digits.
	phoneNumberMinDigits = 7
	phoneNumberMaxDigits = 15
	// phoneNumberIndex is the unique index on users.phone.
	phoneNumberIndex = "idx_users_phone"
	// phoneNumberPunctuation is dropped like spaces and hyphens.
	phoneNumberPunctuation = ".()[]．（）［］"
)

var (
	phoneCountryCodeOption = regexp.MustCompile(`^[1-9][0-9]{0,2}$`)
	phoneTrunkPrefixOption = regexp.MustCompile(`^[0-9]{1,3}$`)
)

// PhoneNumberFormat describes the accepted phone numbers. Without a country
// code any number written in international form is accepted, and the trunk
// prefix and the pattern do not apply.
type PhoneNumberFormat struct {
	CountryCode           string // country calling code, digits without +
	TrunkPrefix           string // written before a national number when it is dialled without the country code
	NationalNumberPattern string // regular expression the whole national number matches
}

// CurrentPhoneNumberFormat is the format the options set.
func CurrentPhoneNumberFormat() PhoneNumberFormat {
	return PhoneNumberFormat{
		CountryCode:           common.PhoneCountryCode,
		TrunkPrefix:           common.PhoneTrunkPrefix,
		NationalNumberPattern: common.PhoneNationalNumberPattern,
	}
}

// Normalize reads a phone number as a person writes it and returns its normal
// form, or the refusal. Digits of any Unicode decimal script are digits;
// spaces, hyphens, dots, brackets and direction marks are dropped; a + may
// only lead.
//
// With a country code the number may be written with + or 00 and the country
// code, with the bare country code, with the trunk prefix, or as the bare
// national number; the readings are tried in that order, the first whose
// national number fits is taken, and the refusal is that of the first reading.
// Without a country code the number must be written with + or 00. The normal
// form is always + and the international digits.
func (format PhoneNumberFormat) Normalize(raw string) (string, error) {
	var digits strings.Builder
	plus := false
	for _, r := range raw {
		switch {
		case r == '+' || r == '＋':
			if plus || digits.Len() > 0 {
				return "", ErrPhoneNumberCharacters
			}
			plus = true
		case unicode.IsDigit(r):
			digits.WriteByte('0' + decimalDigitValue(r))
		case unicode.IsSpace(r), unicode.Is(unicode.Pd, r), unicode.Is(unicode.Bidi_Control, r), strings.ContainsRune(phoneNumberPunctuation, r):
		default:
			return "", ErrPhoneNumberCharacters
		}
	}
	number := digits.String()
	if number == "" && !plus {
		return "", ErrPhoneNumberMissing
	}
	international := plus
	if !plus {
		number, international = strings.CutPrefix(number, "00")
	}

	if format.CountryCode == "" {
		switch {
		case !international:
			return "", ErrPhoneNumberFormat
		case len(number) < phoneNumberMinDigits || len(number) > phoneNumberMaxDigits:
			return "", ErrPhoneNumberLength
		case number[0] == '0':
			return "", ErrPhoneNumberFormat
		}
		return "+" + number, nil
	}

	var readings []string
	if national, ok := strings.CutPrefix(number, format.CountryCode); ok {
		// "+44 (0) 20 ..." writes the trunk prefix after the country code.
		if trimmed, ok := strings.CutPrefix(national, format.TrunkPrefix); ok && format.TrunkPrefix != "" {
			readings = append(readings, trimmed)
		}
		readings = append(readings, national)
	}
	if !international {
		if national, ok := strings.CutPrefix(number, format.TrunkPrefix); ok && format.TrunkPrefix != "" {
			readings = append(readings, national)
		}
		readings = append(readings, number)
	}
	if len(readings) == 0 {
		// An international number of another country.
		return "", ErrPhoneNumberFormat
	}
	var pattern *regexp.Regexp
	if format.NationalNumberPattern != "" {
		var err error
		if pattern, err = regexp.Compile("^(?:" + format.NationalNumberPattern + ")$"); err != nil {
			common.SysError(common.LogText("the national number pattern does not compile: %s", err.Error()))
			return "", ErrPhoneNumberFormat
		}
	}
	var refusal error
	for _, national := range readings {
		var reason error
		switch digitCount := len(format.CountryCode) + len(national); {
		case digitCount < phoneNumberMinDigits || digitCount > phoneNumberMaxDigits:
			reason = ErrPhoneNumberLength
		case pattern != nil && !pattern.MatchString(national):
			reason = ErrPhoneNumberFormat
		default:
			return "+" + format.CountryCode + national, nil
		}
		if refusal == nil {
			refusal = reason
		}
	}
	return "", refusal
}

// Display writes a stored number as the format reads it nationally: the trunk
// prefix and the national number when the number has the format's country
// code, the stored international number otherwise.
func (format PhoneNumberFormat) Display(phone string) string {
	national, ok := strings.CutPrefix(phone, "+"+format.CountryCode)
	if !ok || format.CountryCode == "" {
		return phone
	}
	return format.TrunkPrefix + national
}

// FillPhoneDisplay sets PhoneDisplay for an answer that shows the account.
func (user *User) FillPhoneDisplay() {
	if user.Phone != nil {
		user.PhoneDisplay = CurrentPhoneNumberFormat().Display(*user.Phone)
	}
}

// decimalDigitValue returns the value of a Unicode decimal digit. Unicode
// assigns every decimal digit set as ten consecutive code points from zero to
// nine, so each range of unicode.Nd starts at a zero.
func decimalDigitValue(r rune) byte {
	for _, digits := range unicode.Nd.R16 {
		if r >= rune(digits.Lo) && r <= rune(digits.Hi) {
			return byte((r - rune(digits.Lo)) % 10)
		}
	}
	for _, digits := range unicode.Nd.R32 {
		if r >= rune(digits.Lo) && r <= rune(digits.Hi) {
			return byte((r - rune(digits.Lo)) % 10)
		}
	}
	return 0
}

// ValidatePhoneNumberFormatOption refuses a value of PhoneCountryCode,
// PhoneTrunkPrefix or PhoneNationalNumberPattern that Normalize could not use.
// An empty value is always accepted.
func ValidatePhoneNumberFormatOption(key string, value string) error {
	if value == "" {
		return nil
	}
	switch key {
	case "PhoneCountryCode":
		if !phoneCountryCodeOption.MatchString(value) {
			return common.NewMessage("The country calling code must be one to three digits that do not start with 0")
		}
	case "PhoneTrunkPrefix":
		if !phoneTrunkPrefixOption.MatchString(value) {
			return common.NewMessage("The trunk prefix must be one to three digits")
		}
	case "PhoneNationalNumberPattern":
		if _, err := regexp.Compile(value); err != nil {
			return common.NewMessage("The national number pattern is not a valid regular expression")
		}
	}
	return nil
}

// EnsurePhoneNumberAvailable refuses a normal-form number another account
// holds. A soft-deleted account keeps its number, as the unique index does: a
// number comes free only when its account is deleted for good.
func EnsurePhoneNumberAvailable(phone string, excludeUserID int) error {
	return ensurePhoneNumberAvailableWithTx(DB, phone, excludeUserID)
}

func ensurePhoneNumberAvailableWithTx(tx *gorm.DB, phone string, excludeUserID int) error {
	query := tx.Unscoped().Model(&User{}).Where("phone = ?", phone)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPhoneNumberTaken
	}
	return nil
}

// phoneNumberConflict turns the unique index refusing a write, when another
// account took the number between the check and the write, into
// ErrPhoneNumberTaken. MySQL and PostgreSQL name the index in the error, SQLite
// names the column.
func phoneNumberConflict(err error) error {
	if err != nil && (strings.Contains(err.Error(), phoneNumberIndex) || strings.Contains(err.Error(), "users.phone")) {
		return ErrPhoneNumberTaken
	}
	return err
}

// RetryPhoneNumberRace makes at most phoneNumberRaceRuns runs. Each deadlock
// rolls back one transaction and lets the others finish, so every run settles
// at least one of the racing sign-ups; a short random pause keeps the
// rolled-back ones from starting again in step. A lock wait timeout already
// waited out the server's whole timeout, so it is retried only once.
const (
	phoneNumberRaceRuns     = 5
	phoneNumberLockWaitRuns = 2
)

// RetryPhoneNumberRace runs insert, a transaction that creates user, again
// when MySQL ends it with a deadlock (1213) or a lock wait timeout (1205) and
// user carries a phone number. A sign-up locks its email before it writes, and
// on MySQL sign-ups racing for one number can deadlock on those locks. A later
// run starts from the same values, waits for the winner to commit, and is
// refused as in use by the same check and unique index as on SQLite and
// PostgreSQL. When the runs are spent, or ctx ends between two runs, the
// answer is ErrPhoneNumberBusy, never the database's text; the transactions
// rolled back, so no account was made and nothing was spent. Any other error,
// and every insert without a number, stays as it was.
func RetryPhoneNumberRace(ctx context.Context, user *User, insert func() error) error {
	values := *user
	lockWaits := 0
	for run := range phoneNumberRaceRuns {
		if run > 0 {
			select {
			case <-ctx.Done():
				return ErrPhoneNumberBusy
			case <-time.After(time.Duration(10+rand.IntN(40*run)) * time.Millisecond):
			}
		}
		*user = values
		err := insert()
		var mysqlErr *mysql.MySQLError
		if values.Phone == nil || !errors.As(err, &mysqlErr) {
			return err
		}
		switch mysqlErr.Number {
		case 1213:
		case 1205:
			if lockWaits++; lockWaits == phoneNumberLockWaitRuns {
				return ErrPhoneNumberBusy
			}
		default:
			return err
		}
	}
	return ErrPhoneNumberBusy
}
