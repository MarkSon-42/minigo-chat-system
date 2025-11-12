package main

import (
	"encoding/json"
	"log"
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

func (ps *ProxyServer) getRules(w http.ResponseWriter, r *http.Request) {
	rules := ps.filter.GetRules()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rules); err != nil {
		log.Printf("[API] Failed to encode rules: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("[API] Returned %d filter rules", len(rules))
}

func (ps *ProxyServer) addRule() {

}
func (ps *ProxyServer) removeRule() {

}
func (ps *ProxyServer) updateRule() {

}
