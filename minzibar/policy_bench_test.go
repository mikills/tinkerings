package main

import (
	"strconv"
	"strings"
	"testing"
)

// lower-case comment: benchmark simple allow/deny policy evaluation
func BenchmarkPolicy_Evaluate_Simple(b *testing.B) {
	policy := `allow if user.department == "engineering" and action == "read"`
	engine := NewPolicy(policy)
	ctx := map[string]string{
		"user.department": "engineering",
		"action":          "read",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(ctx)
	}
}

// lower-case comment: benchmark evaluation with multiple rules and or/and logic
func BenchmarkPolicy_Evaluate_Complex(b *testing.B) {
	policy := `
		allow if (user.department == "engineering" and action == "read") or
		          (user.department == "qa" and action == "test") or
		          (user.department == "finance" and action == "audit") or
		          (user.department == "hr" and action == "review") or
		          (user.department == "ops" and action == "deploy")
		deny if user.role == "contractor" and action == "delete"
	`
	engine := NewPolicy(policy)
	ctx := map[string]string{
		"user.department": "finance",
		"action":          "audit",
		"user.role":       "employee",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(ctx)
	}
}

// lower-case comment: benchmark evaluation with a large number of rules
func BenchmarkPolicy_Evaluate_ManyRules(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("allow if user.department == \"d" + strconv.Itoa(i) + "\" and action == \"a" + strconv.Itoa(i) + "\"\n")
	}
	sb.WriteString("deny if user.role == \"contractor\" and action == \"delete\"\n")
	engine := NewPolicy(sb.String())
	ctx := map[string]string{
		"user.department": "d77",
		"action":          "a77",
		"user.role":       "employee",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(ctx)
	}
}

// lower-case comment: benchmark evaluation with a rule that never matches (worst-case scan)
func BenchmarkPolicy_Evaluate_NoMatch(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("allow if user.department == \"d" + strconv.Itoa(i) + "\" and action == \"a" + strconv.Itoa(i) + "\"\n")
	}
	engine := NewPolicy(sb.String())
	ctx := map[string]string{
		"user.department": "notfound",
		"action":          "none",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(ctx)
	}
}
