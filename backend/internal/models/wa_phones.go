package models

import (
	"encoding/json"
	"strings"
	"unicode"
)

const MaxWAAllowedPhones = 50

func NormalizeWAPhoneDigits(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if strings.HasPrefix(digits, "0") {
		digits = "62" + strings.TrimPrefix(digits, "0")
	}
	return digits
}

func ParseWAAllowedPhones(stored *string) []string {
	if stored == nil {
		return nil
	}
	s := strings.TrimSpace(*stored)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		var phones []string
		if err := json.Unmarshal([]byte(s), &phones); err == nil {
			return compactWAPhones(phones)
		}
	}
	return compactWAPhones(strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	}))
}

func JoinWAAllowedPhones(phones []string) *string {
	phones = compactWAPhones(phones)
	if len(phones) == 0 {
		return nil
	}
	b, err := json.Marshal(phones)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

func compactWAPhones(phones []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(phones))
	for _, p := range phones {
		n := NormalizeWAPhoneDigits(p)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
		if len(out) >= MaxWAAllowedPhones {
			break
		}
	}
	return out
}

func (i *Integration) HydrateWAAllowed(raw *string) {
	if i == nil {
		return
	}
	i.WAAllowedPhones = ParseWAAllowedPhones(raw)
}

func (i *Integration) StoredWAAllowedPhones() *string {
	if i == nil {
		return nil
	}
	return JoinWAAllowedPhones(i.WAAllowedPhones)
}

func (i *Integration) SenderAllowed(senderDigits string) bool {
	return i.AnySenderAllowed([]string{senderDigits})
}

func (i *Integration) AnySenderAllowed(senders []string) bool {
	if i == nil || len(i.WAAllowedPhones) == 0 {
		return true
	}
	for _, s := range senders {
		s = NormalizeWAPhoneDigits(s)
		if s == "" {
			continue
		}
		for _, p := range i.WAAllowedPhones {
			if waPhonesMatch(p, s) {
				return true
			}
		}
	}
	return false
}

func waPhonesMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	a62 := strings.TrimPrefix(a, "62")
	b62 := strings.TrimPrefix(b, "62")
	return a == b62 || b == a62 || a62 == b62
}
