package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Service wraps the Engine and exposes HTTP endpoints.
type Service struct {
	Engine *Engine
}

// NewService creates a new Service with the given Engine.
func NewService(engine *Engine) *Service {
	return &Service{Engine: engine}
}

// Run starts the Echo server and registers routes.
func (s *Service) Run(addr string) error {
	e := echo.New()

	// create resource
	e.POST("/resource", s.handleCreateResource)
	// add relation via query
	e.POST("/relation", s.handleAddRelationQuery)
	// add policy
	e.POST("/policy", s.handleAddPolicy)
	// attach policy to resource
	e.POST("/policy/attach", s.handleAttachPolicy)
	// verify access
	e.POST("/verify", s.handleVerify)
	// list all resources
	e.GET("/objects", s.handleListAllResources)

	return e.Start(addr)
}

// --- Handlers ---

type CreateResourceRequest struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (s *Service) handleCreateResource(c echo.Context) error {
	var req CreateResourceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	obj := s.Engine.CreateResource(req.Type, req.ID)
	return c.JSON(http.StatusOK, obj)
}

type CreateSubjectRequest struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Relation string `json:"relation"`
}

type AddRelationQueryRequest struct {
	Query string `json:"query"`
}

func (s *Service) handleAddRelationQuery(c echo.Context) error {
	var req AddRelationQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.Engine.AddRelationQuery(req.Query); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "relation(s) added"})
}

type AddPolicyRequest struct {
	PolicyID   string `json:"policy_id"`
	PolicyText string `json:"policy_text"`
}

func (s *Service) handleAddPolicy(c echo.Context) error {
	var req AddPolicyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := s.Engine.AddPolicy(req.PolicyID, req.PolicyText); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "policy added"})
}

type AttachPolicyRequest struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	PolicyID     string `json:"policy_id"`
}

func (s *Service) handleAttachPolicy(c echo.Context) error {
	var req AttachPolicyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	resource := ObjectRef{Type: req.ResourceType, ObjectID: req.ResourceID}
	if err := s.Engine.AddPolicyToResource(resource, req.PolicyID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "policy attached"})
}

type VerifyRequest struct {
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	SubjectType  string            `json:"subject_type"`
	SubjectID    string            `json:"subject_id"`
	Action       string            `json:"action"`
	Context      map[string]string `json:"context"`
}

func (s *Service) handleVerify(c echo.Context) error {
	var req VerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	resource := ObjectRef{Type: req.ResourceType, ObjectID: req.ResourceID}
	subject := ObjectRef{Type: req.SubjectType, ObjectID: req.SubjectID}
	allowed, err := s.Engine.Verify(resource, subject, req.Action, req.Context)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"allowed": allowed})
}

func (s *Service) handleListAllResources(c echo.Context) error {
	objects := s.Engine.ListAllResources()
	return c.JSON(http.StatusOK, objects)
}
