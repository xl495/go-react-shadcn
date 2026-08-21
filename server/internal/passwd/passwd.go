package passwd

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrTooShort   = errors.New("password must be at least 8 characters")
	ErrSameAsUser = errors.New("password must not match username")
	ErrWeak       = errors.New("password must include letters and digits")
	ErrSeed       = errors.New("default seed password is not allowed")
)

var dummyHash string

func init() {
	hash, err := Hash("latch-dummy-password-not-a-user")
	if err != nil {
		panic(err)
	}
	dummyHash = hash
}

func DummyHash() string {
	return dummyHash
}

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Match(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func Check(plain, username string) error {
	if utf8.RuneCountInString(plain) < 8 {
		return ErrTooShort
	}
	if username != "" && strings.EqualFold(plain, username) {
		return ErrSameAsUser
	}
	var letter, digit bool
	for _, r := range plain {
		if unicode.IsLetter(r) {
			letter = true
		}
		if unicode.IsDigit(r) {
			digit = true
		}
	}
	if !letter || !digit {
		return ErrWeak
	}
	return nil
}

func CheckProduction(plain, username string) error {
	if err := Check(plain, username); err != nil {
		return err
	}
	var upper, lower bool
	for _, r := range plain {
		if unicode.IsUpper(r) {
			upper = true
		}
		if unicode.IsLower(r) {
			lower = true
		}
	}
	if IsBuiltinSeedPassword(plain) {
		return ErrSeed
	}
	if !upper || !lower {
		return ErrWeak
	}
	return nil
}

func IsBuiltinSeedPassword(plain string) bool {
	switch plain {
	case "admin123", "viewer123", "operator123", "webuser123":
		return true
	default:
		return false
	}
}
