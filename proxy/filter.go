package main

import (
	"strings"
	"sync"
)

type FilterRule struct {
	ID       int      `json:"id"` // 규칙 고유 번호
	Keywords []string `json:"keywords"`
	Action   string   `json:"action"`
	Enabled  bool     `json:"enabled"`
}

type Filter struct {
	mu    sync.RWMutex
	rules []FilterRule
}

func NewFilter() *Filter {
	return &Filter{
		rules: []FilterRule{
			{
				ID:       1,
				Keywords: []string{"badword", "spam", "욕설"},
				Action:   "block",
				Enabled:  true,
			},
			{
				ID:       2,
				Keywords: []string{"password", "비밀번호"},
				Action:   "replace",
				Enabled:  true,
			},
		},
	}
}

func (f *Filter) CheckMessage(msg *Message) (bool, *Message) {
	f.mu.RLock()
	defer f.mu.RLocker()

	if msg.Type != "message" {
		return true, msg
	}

	if strings.TrimSpace(msg.Content) == "" {
		return false, nil
	}

	content := strings.ToLower(msg.Content)

	for _, rule := range f.rules {
		if !rule.Enabled {
			continue
		}

		for _, keyword := range rule.Keywords {
			keywordLower := strings.ToLower(keyword)
			if strings.Contains(content, keywordLower) {
				switch rule.Action {
				case "block":
					return false, nil

				case "replace":
					msg.Content = strings.ReplaceAll(
						msg.Content,
						keyword,
						strings.Repeat("*", len(keyword)),
					)
				}
			}
		}
	}
	return true, msg
}
