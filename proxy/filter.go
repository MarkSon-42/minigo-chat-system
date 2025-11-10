package main

import (
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
