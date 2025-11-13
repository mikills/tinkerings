package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestService_FeatureFlagAccessFromMockData(t *testing.T) {
	// Load mock_data.json
	data, err := os.ReadFile("mock_data.json")
	if err != nil {
		t.Fatalf("failed to read mock_data.json: %v", err)
	}

	// Parse mock data
	var mock struct {
		FeatureFlag struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"feature_flag"`
		Users []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Department string `json:"department"`
		} `json:"users"`
		Relations []struct {
			ResourceType string `json:"resource_type"`
			ResourceID   string `json:"resource_id"`
			Relation     string `json:"relation"`
			SubjectType  string `json:"subject_type"`
			SubjectID    string `json:"subject_id"`
		} `json:"relations"`
		Policy struct {
			PolicyID   string `json:"policy_id"`
			PolicyText string `json:"policy_text"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(data, &mock); err != nil {
		t.Fatalf("failed to parse mock_data.json: %v", err)
	}

	engine := NewEngine(NewRelationGraph(), map[string]*Policy{})
	service := NewService(engine)
	e := echo.New()

	// Create feature flag resource
	{
		reqBody, _ := json.Marshal(map[string]string{
			"type": mock.FeatureFlag.Type,
			"id":   mock.FeatureFlag.ID,
		})
		req := httptest.NewRequest(http.MethodPost, "/resource", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := service.handleCreateResource(c); err != nil {
			t.Fatalf("create resource failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("create resource failed: status %d", rec.Code)
		}
	}

	// Only use AddRelationQuery for users with access (those in mock.Relations)
	userDept := make(map[string]string)
	for _, u := range mock.Users {
		userDept[u.ID] = u.Department
	}
	for _, rel := range mock.Relations {
		query := rel.ResourceType + ":" + rel.ResourceID + " " + rel.SubjectType + ":" + rel.SubjectID + "->" + rel.Relation
		reqBody, _ := json.Marshal(map[string]string{
			"query": query,
		})
		req := httptest.NewRequest(http.MethodPost, "/relation", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := service.handleAddRelationQuery(c); err != nil {
			t.Fatalf("add relation failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("add relation failed: status %d", rec.Code)
		}
	}

	// Add policy
	{
		reqBody, _ := json.Marshal(map[string]string{
			"policy_id":   mock.Policy.PolicyID,
			"policy_text": mock.Policy.PolicyText,
		})
		req := httptest.NewRequest(http.MethodPost, "/policy", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := service.handleAddPolicy(c); err != nil {
			t.Fatalf("add policy failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("add policy failed: status %d", rec.Code)
		}
	}

	// Attach policy
	{
		reqBody, _ := json.Marshal(map[string]string{
			"resource_type": mock.FeatureFlag.Type,
			"resource_id":   mock.FeatureFlag.ID,
			"policy_id":     mock.Policy.PolicyID,
		})
		req := httptest.NewRequest(http.MethodPost, "/policy/attach", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := service.handleAttachPolicy(c); err != nil {
			t.Fatalf("attach policy failed: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("attach policy failed: status %d", rec.Code)
		}
	}

	// Verify access for all users
	for _, u := range mock.Users {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"resource_type": mock.FeatureFlag.Type,
			"resource_id":   mock.FeatureFlag.ID,
			"subject_type":  u.Type,
			"subject_id":    u.ID,
			"action":        "enabled",
			"context":       map[string]string{"department": u.Department},
		})
		req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := service.handleVerify(c); err != nil {
			t.Errorf("verify failed for user %s: %v", u.ID, err)
			continue
		}

		if rec.Code != http.StatusOK {
			t.Errorf("verify failed for user %s: status %d", u.ID, rec.Code)
			continue
		}

		var resp struct {
			Allowed bool `json:"allowed"`
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("verify response unmarshal failed for user %s: %v", u.ID, err)
			continue
		}

		shouldHaveAccess := u.Department == "CTO" && (u.ID == "user1" || u.ID == "user2" || u.ID == "user3" || u.ID == "user4" || u.ID == "user5")
		if resp.Allowed != shouldHaveAccess {
			t.Errorf("user %s (dept: %s) access mismatch: got %v, want %v",
				u.ID, u.Department, resp.Allowed, shouldHaveAccess)
		}
	}
}
