package spec

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func acceptance(t *testing.T, name string) Acceptance {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return ParseAcceptance(bytes.NewReader(b))
}

func TestParseAcceptance(t *testing.T) {
	cases := []struct {
		file string
		ref  string
		amb  int
	}{
		{"acceptance_en.md", "1a2b3c4", 1},
		{"acceptance_fr.md", "9f8e7d6c5b", 1},
		{"acceptance_clean.md", "deadbeef", 0},
	}
	for _, c := range cases {
		got := acceptance(t, c.file)
		if got.Reference != c.ref || got.Ambiguities != c.amb {
			t.Errorf("%s: got %+v", c.file, got)
		}
	}
}

func TestParseAcceptanceEmpty(t *testing.T) {
	if got := ParseAcceptance(strings.NewReader("")); got != (Acceptance{}) {
		t.Errorf("got %+v", got)
	}
	if got := ParseAcceptance(strings.NewReader("Reference: nothex\n")); got.Reference != "" {
		t.Errorf("non hex reference accepted: %+v", got)
	}
}

func verification(t *testing.T, name string) Verification {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return ParseVerification(bytes.NewReader(b))
}

func TestParseVerification(t *testing.T) {
	got := verification(t, "verification_pass.md")
	if got.Verdict != "PASS" || got.Partial != 1 || got.Missing != 0 ||
		!reflect.DeepEqual(got.OK, []string{"AC-1 Log in", "AC-2 Log out"}) {
		t.Errorf("pass: %+v", got)
	}
	got = verification(t, "verification_fail.md")
	if got.Verdict != "FAIL" || got.Missing != 2 || !reflect.DeepEqual(got.OK, []string{"C-1 Export CSV"}) {
		t.Errorf("fail: %+v", got)
	}
}

func TestParseVerificationEmpty(t *testing.T) {
	if got := ParseVerification(strings.NewReader("no report here")); got.Verdict != "" || len(got.OK) != 0 {
		t.Errorf("got %+v", got)
	}
}
