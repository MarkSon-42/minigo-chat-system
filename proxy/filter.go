package main

import (
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis/passes/defers"
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
	defer f.mu.RUnlock()

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

func (f *Filter) AddRule(rule FilterRule) {
	f.mu.Lock() // 쓰기 락 (배타적 접근)
	defer f.mu.Unlock()

	// 새 ID 자동 할당 (기존 최대 ID + 1)
	maxID := 0
	for _, r := range f.rules {
		if r.ID > maxID {
			maxID = r.ID
		}
	}
	rule.ID = maxID + 1

	f.rules = append(f.rules, rule)
}

func (f *Filter) RemoveRule(id int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, rule := range f.rules {
		if rule.ID == id {
			f.rules = append(f.rules[:i], f.rules[i+1:]...)
			return true
		}
	}
	return false
}

func (f *Filter) UpdateRule(id int, new Rule FilterRule) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, rule := range f.rules {
		if rule.ID == id {
			newRule.ID = id
			f.rules[i] = newRule
			return true
		}
	}
	return false
}

func (f *Filter) GetRules() []FilterRule {
	f.mu.RLock()
	defer f.mu.RUnlock()

	rules := make([]FilterRule, len(f.rules))
	copy(rules, f.rules)
	return rules
}


