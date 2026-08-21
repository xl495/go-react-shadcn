package passwd

import "testing"

func TestCheckRequiresLetterAndDigit(t *testing.T) {
	if err := Check("short", "u"); err != ErrTooShort {
		t.Fatalf("short: %v", err)
	}
	if err := Check("username1", "username1"); err != ErrSameAsUser {
		t.Fatalf("same: %v", err)
	}
	if err := Check("onlyletters", "u"); err != ErrWeak {
		t.Fatalf("letters: %v", err)
	}
	if err := Check("pass-1234", "u"); err != nil {
		t.Fatalf("ok: %v", err)
	}
}

func TestCheckProductionRejectsSeedAndNeedsCase(t *testing.T) {
	if err := CheckProduction("admin123", "admin"); err != ErrSeed {
		t.Fatalf("seed: %v", err)
	}
	if err := CheckProduction("password1", "admin"); err != ErrWeak {
		t.Fatalf("case: %v", err)
	}
	if err := CheckProduction("Password1", "admin"); err != nil {
		t.Fatalf("ok: %v", err)
	}
}

func TestDummyHashMatchesWithoutPanic(t *testing.T) {
	if DummyHash() == "" {
		t.Fatal("empty dummy hash")
	}
	if Match(DummyHash(), "not-the-dummy") {
		t.Fatal("dummy should not match random secret")
	}
}
