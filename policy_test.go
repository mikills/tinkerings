package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// context for evaluation: user, resource, action, etc.
type evalContext map[string]string

func TestPolicy_AllowSimple(t *testing.T) {
	policy := `allow read if user.department == "engineering"`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.department": "engineering",
		"action":          "read",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.department": "engineering",
		"action":          "write",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

func TestPolicy_DenyRule(t *testing.T) {
	policy := `
		allow read if user.department == "engineering"
		deny delete if user.role == "contractor"
	`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.department": "engineering",
		"user.role":       "contractor",
		"action":          "delete",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)

	ctx = evalContext{
		"user.department": "engineering",
		"user.role":       "developer",
		"action":          "read",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)
}

func TestPolicy_OrLogic(t *testing.T) {
	policy := `
		allow * if user.department == "engineering" or user.department == "qa"
	`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.department": "engineering",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.department": "qa",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.department": "finance",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

func TestPolicy_NotLogic(t *testing.T) {
	policy := `
		allow * if not user.role == "contractor"
	`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.role": "contractor",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)

	ctx = evalContext{
		"user.role": "developer",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)
}

func TestPolicy_DefaultDeny(t *testing.T) {
	policy := `
		allow * if user.department == "engineering"
	`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.department": "finance",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

func TestPolicyBuilder_Build(t *testing.T) {
	policy := `
		allow read if user.department == "engineering"
		deny delete if user.role == "contractor"
	`
	builder, err := NewPolicyBuilder(policy)
	assert.NoError(t, err)
	engine := builder.Build()

	ctx := evalContext{
		"user.department": "engineering",
		"action":          "read",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.role": "contractor",
		"action":    "delete",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

func TestPolicy_ChainedOrAnd(t *testing.T) {
	policy := `
		allow read if user.department == "engineering"
		allow test if user.department == "qa"
		allow audit if user.department == "finance"
		allow review if user.department == "hr"
		allow deploy if user.department == "ops"
	`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.department": "finance",
		"action":          "audit",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.department": "legal",
		"action":          "review",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

// test for '!=' operator in policy rule
func TestPolicy_NotEqualOperator(t *testing.T) {
	policy := `allow read if user.department != "engineering"`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.department": "finance",
		"action":          "read",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.department": "engineering",
		"action":          "read",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

// test for contains() keyword in policy rule
func TestPolicy_ContainsKeyword(t *testing.T) {
	policy := `allow read if contains(user.groups, "admin")`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.groups": "admin,engineering,qa",
		"action":      "read",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.groups": "engineering,qa",
		"action":      "read",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}

// test for all() keyword in policy rule
func TestPolicy_AllKeyword(t *testing.T) {
	policy := `allow read if all(user.groups, "admin")`
	engine := NewPolicy(policy)

	ctx := evalContext{
		"user.groups": "admin,admin",
		"action":      "read",
	}
	result := engine.Evaluate(ctx)
	assert.Equal(t, "allow", result)

	ctx = evalContext{
		"user.groups": "admin,engineering",
		"action":      "read",
	}
	result = engine.Evaluate(ctx)
	assert.Equal(t, "deny", result)
}
