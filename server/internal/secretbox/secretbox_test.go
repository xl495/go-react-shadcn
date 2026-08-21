package secretbox

import "testing"

func TestSealOpenRoundTrip(t *testing.T) {
	got, err := Seal("master-key", "smtp-password")
	if err != nil {
		t.Fatal(err)
	}
	if got == "smtp-password" || !IsSealed(got) {
		t.Fatalf("expected sealed, got %q", got)
	}
	plain, err := Open("master-key", got)
	if err != nil || plain != "smtp-password" {
		t.Fatalf("open: %q %v", plain, err)
	}
	if _, err := Open("other-key", got); err == nil {
		t.Fatal("wrong key should fail")
	}
	if MustOpen("other-key", got) != "" {
		t.Fatal("MustOpen should return empty on decrypt failure")
	}
	out, err := Open("x", "plain-secret")
	if err != nil || out != "plain-secret" {
		t.Fatalf("plaintext passthrough: %q %v", out, err)
	}
}
