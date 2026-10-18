package customer

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzCustomerFieldValidators exercises the name/email/phone validators shared
// by customer create and update. The contract is one of *shape*: an accepted
// value must satisfy the documented bounds, and a rejected value must violate
// one of them. Each accepted value is re-checked against an independently
// written version of those bounds, so a change to the regexes or limits that
// lets a shape through fails here.
func FuzzCustomerFieldValidators(f *testing.F) {
	f.Add("")
	f.Add("   ")
	f.Add("Jane Doe")
	f.Add("jane.doe+orders@example.co.id")
	f.Add("jane@example")
	f.Add("jane@@example.com")
	f.Add("+62 812-3456-7890")
	f.Add("0812 3456")
	f.Add("081234567890123456789")
	f.Add(strings.Repeat("a", 201))
	f.Add("josé@example.com")

	f.Fuzz(func(t *testing.T, s string) {
		nameErr := validateCustomerName(s)
		emailErr := validateCustomerEmail(s)
		phoneErr := validateCustomerPhone(s)

		blank := strings.TrimSpace(s) == ""
		if blank && (nameErr == nil || emailErr == nil || phoneErr == nil) {
			t.Fatalf("blank value %q accepted (name=%v email=%v phone=%v)", s, nameErr, emailErr, phoneErr)
		}

		if nameErr == nil {
			if blank || len(s) > 200 {
				t.Fatalf("validateCustomerName(%q) accepted a value outside 1..200 bytes", s)
			}
		} else if !blank && len(s) <= 200 {
			t.Fatalf("validateCustomerName(%q) rejected a value within 1..200 bytes", s)
		}

		if emailErr == nil {
			if s != strings.TrimSpace(s) {
				t.Fatalf("validateCustomerEmail(%q) accepted surrounding whitespace", s)
			}
			at := strings.IndexByte(s, '@')
			if at <= 0 || at != strings.LastIndexByte(s, '@') || at == len(s)-1 {
				t.Fatalf("validateCustomerEmail(%q) accepted a value without exactly one interior @", s)
			}
			domain := s[at+1:]
			dot := strings.LastIndexByte(domain, '.')
			if dot <= 0 || dot == len(domain)-1 {
				t.Fatalf("validateCustomerEmail(%q) accepted a domain without an interior dot", s)
			}
			if suffix := domain[dot+1:]; len(suffix) < 2 {
				t.Fatalf("validateCustomerEmail(%q) accepted top-level domain %q shorter than two characters", s, suffix)
			} else {
				for _, r := range suffix {
					if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
						t.Fatalf("validateCustomerEmail(%q) accepted non-alphabetic top-level domain %q", s, suffix)
					}
				}
			}
		}

		if phoneErr == nil {
			if n := utf8.RuneCountInString(s); n < 7 || n > 20 {
				t.Fatalf("validateCustomerPhone(%q) accepted %d runes, want 7..20", s, n)
			}
			for _, r := range s {
				if !strings.ContainsRune("0123456789+-() ", r) {
					t.Fatalf("validateCustomerPhone(%q) accepted disallowed rune %q", s, r)
				}
			}
		}
	})
}
