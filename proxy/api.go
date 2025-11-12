package main

import (
	"net/http"
)

func (ps *ProxyServer) handleFilterRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ps.getRules(w, r)
	case http.MethodPost:
		ps.addRule(w, r)
	case http.MethodDelete:
		ps.removeRule(w, r)
	case http.MethodPut:
		ps.updateRule(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
