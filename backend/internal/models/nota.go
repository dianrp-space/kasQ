package models

import (
	"encoding/json"
	"strings"
)

const MaxNotaFiles = 10

func ParseNotaKeys(stored *string) []string {
	if stored == nil {
		return nil
	}
	s := strings.TrimSpace(*stored)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		var keys []string
		if err := json.Unmarshal([]byte(s), &keys); err == nil {
			return compactNotaKeys(keys)
		}
	}
	return []string{s}
}

func JoinNotaKeys(keys []string) *string {
	keys = compactNotaKeys(keys)
	if len(keys) == 0 {
		return nil
	}
	if len(keys) == 1 {
		return &keys[0]
	}
	b, err := json.Marshal(keys)
	if err != nil {
		return &keys[0]
	}
	s := string(b)
	return &s
}

func compactNotaKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

func (t *Transaction) HydrateNota() {
	if t == nil {
		return
	}
	if t.notaRaw == nil && t.NotaKey != nil {
		raw := *t.NotaKey
		t.notaRaw = &raw
	}
	t.NotaKeys = ParseNotaKeys(t.StoredNota())
	if len(t.NotaKeys) > 0 {
		first := t.NotaKeys[0]
		t.NotaKey = &first
	} else {
		t.NotaKey = nil
	}
}

func (t *Transaction) StoredNota() *string {
	if t == nil {
		return nil
	}
	if t.notaRaw != nil {
		return t.notaRaw
	}
	return t.NotaKey
}
