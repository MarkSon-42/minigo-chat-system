package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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

func (ps *ProxyServer) addRule(w http.ResponseWriter, r *http.Request) {
	var rule FilterRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		log.Printf("[API] Invalid request body: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(rule.Keywords) == 0 {
		http.Error(w, "Keywords cannot be empty", http.StatusBadRequest)
		return
	}

	if rule.Action != "block" && rule.Action != "replace" {
		http.Error(w, "Action must be 'block' or 'replace'", http.StatusBadRequest)
		return
	}

	if rule.Action == "replace" && rule.Replacement == "" {
		http.Error(w, "Replacement text required for 'replace' action", http.StatusBadRequest)
		return
	}

	id := ps.filter.AddRule(rule)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "created",
		"message": "Filter rule added successfully",
		"id":      id,
	}); err != nil {
		log.Printf("[API] Failed to encode response: %v", err)
	}

	log.Printf("[API] Added new rule ID=%d: %v", id, rule.Keywords)
}

func (ps *ProxyServer) removeRule(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid 'id' parameter", http.StatusBadRequest)
		return
	}

	if !ps.filter.RemoveRule(id) {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "deleted",
		"message": "Filter rule removed successfully",
	}); err != nil {
		log.Printf("[API] Failed to encode response: %v", err)
	}

	log.Printf("[API] Removed rule ID=%d", id)
}

func (ps *ProxyServer) updateRule(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid 'id' parameter", http.StatusBadRequest)
		return
	}

	var newRule FilterRule
	if err := json.NewDecoder(r.Body).Decode(&newRule); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(newRule.Keywords) == 0 {
		http.Error(w, "Keywords cannot be empty", http.StatusBadRequest)
		return
	}

	if newRule.Action != "block" && newRule.Action != "replace" {
		http.Error(w, "Action must be 'block' or 'replace'", http.StatusBadRequest)
		return
	}

	if !ps.filter.UpdateRule(id, newRule) {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "updated",
		"message": "Filter rule updated successfully",
	}); err != nil {
		log.Printf("[API] Failed to encode response: %v", err)
	}

	log.Printf("[API] Updated rule ID=%d", id)
}
