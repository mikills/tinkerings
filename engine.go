package main

import (
	"fmt"
	"strings"
)

// engine is the main policy engine struct and implements the asserter interface.
type Engine struct {
	graph      *RelationGraph
	policyRepo map[string]*Policy // policyID -> Policy

	// TODO: add stubs map for frequent resource creation [user/group/orgs/featur_flag]
	// That way someone can just  select the stab and create on.
	// so select feature flag and produce a feature_name and then the resource will be "feature_flag_feature_name"
}

// addpolicy attaches a policy to a resource via the relation graph
func (e *Engine) AddPolicyToResource(resource ObjectRef, policyID string) error {
	// validate resource existence: must have at least one relation tuple
	exists := false
	for _, t := range e.graph.ReadTuples(resource, "") {
		if t.Object == resource {
			exists = true
			break
		}
	}
	if !exists {
		return fmt.Errorf("resource %s does not exist in graph", resource.String())
	}
	tuple := RelationTuple{
		Object:   resource,
		Relation: "has_policy",
		Subject:  SubjectRef{Object: ObjectRef{Type: "policy", ObjectID: policyID}},
	}
	e.graph.Write(tuple)
	return nil
}

// addpolicy registers a new policy in the policyRepo
func (e *Engine) AddPolicy(policyID string, policyText string) error {
	policy := NewPolicy(policyText)
	e.policyRepo[policyID] = policy
	return nil
}

// createresource returns an objectref for a new resource and adds a marker relation to the graph
func (e *Engine) CreateResource(resourceType, resourceID string) ObjectRef {
	obj := ObjectRef{Type: resourceType, ObjectID: resourceID}
	// add a marker relation so resource exists in graph
	tuple := RelationTuple{
		Object:   obj,
		Relation: "resource",
		Subject:  SubjectRef{Object: ObjectRef{Type: "system", ObjectID: "resource_marker"}},
	}
	e.graph.Write(tuple)
	return obj
}

// createsubject returns a subjectref for a new subject/userset
func (e *Engine) CreateSubject(subjectType, subjectID string, relation string) SubjectRef {
	obj := ObjectRef{Type: subjectType, ObjectID: subjectID}
	return SubjectRef{Object: obj, Relation: relation}
}

// addrelation adds a relationship tuple to the graph
func (e *Engine) AddRelation(object ObjectRef, relation string, subject SubjectRef) error {
	tuple := RelationTuple{
		Object:   object,
		Relation: relation,
		Subject:  subject,
	}
	e.graph.Write(tuple)
	return nil
}

// removerelation removes a relationship tuple from the graph
func (e *Engine) RemoveRelation(object ObjectRef, relation string, subject SubjectRef) error {
	tuple := RelationTuple{
		Object:   object,
		Relation: relation,
		Subject:  subject,
	}
	if !e.graph.Delete(tuple) {
		return fmt.Errorf("relation does not exist")
	}
	return nil
}

// getrelation returns all relations between object and subject
func (e *Engine) GetRelation(object ObjectRef, subject SubjectRef) []string {
	var relations []string
	for _, t := range e.graph.ReadTuples(object, "") {
		if t.Subject == subject {
			relations = append(relations, t.Relation)
		}
	}
	return relations
}

// checkrelation checks if a specific relation exists between object and subject
func (e *Engine) CheckRelation(object ObjectRef, relation string, subject SubjectRef) bool {
	return e.graph.HasDirectRelation(object, relation, subject)
}

// getresources returns all resources a subject has a given relation to
func (e *Engine) GetResources(subject ObjectRef, relation string) []ObjectRef {
	return e.graph.GetObjects(subject, relation)
}

// addrelationquery parses a query string and adds relations for the subject
// query format: "document:doc123 user:alice->read,write"
func (e *Engine) AddRelationQuery(query string) error {
	parts := strings.Fields(query)
	if len(parts) < 2 {
		return fmt.Errorf("invalid query format: must include resource and at least one subject->action pair")
	}
	resourceStr := parts[0]
	resource, err := parseObjectRef(resourceStr)
	if err != nil {
		return fmt.Errorf("invalid resource: %v", err)
	}
	// validate resource existence: must have at least one tuple
	exists := false
	for _, t := range e.graph.ReadTuples(resource, "") {
		if t.Object == resource {
			exists = true
			break
		}
	}
	if !exists {
		return fmt.Errorf("resource %s does not exist in graph", resource.String())
	}
	for _, subPerm := range parts[1:] {
		parsed, err := ParseSubjectPermissions(subPerm)
		if err != nil {
			return fmt.Errorf("invalid subject/action pair: %v", err)
		}
		for _, action := range parsed.Actions {
			e.graph.Write(RelationTuple{
				Object:   resource,
				Relation: action,
				Subject:  parsed.Subject,
			})
		}
	}
	return nil
}

// getobjects returns all subjects that have a given relation to an object
func (e *Engine) GetObjects(object ObjectRef, relation string) []ObjectRef {
	subjectRefs := e.graph.GetSubjects(object, relation)
	var objs []ObjectRef
	for _, s := range subjectRefs {
		objs = append(objs, s.Object)
	}
	return objs
}

// listallobjects returns all ObjectRef instances referenced in the graph
func (e *Engine) ListAllResources() []ObjectRef {
	return e.graph.ListAllObjects()
}

// deletepolicy detaches a policy from a resource via the relation graph
func (e *Engine) DeletePolicy(resource ObjectRef, policyID string) error {
	tuple := RelationTuple{
		Object:   resource,
		Relation: "has_policy",
		Subject:  SubjectRef{Object: ObjectRef{Type: "policy", ObjectID: policyID}},
	}
	e.graph.Delete(tuple)
	return nil
}

// getpolicies returns all policies attached to a resource
func (e *Engine) GetPolicies(resource ObjectRef) ([]*Policy, error) {
	tuples := e.graph.ReadTuples(resource, "has_policy")
	var policies []*Policy
	for _, t := range tuples {
		pid := t.Subject.Object.ObjectID
		if p, ok := e.policyRepo[pid]; ok {
			policies = append(policies, p)
		}
	}
	return policies, nil
}

// verify checks if a subject has access to a resource for a given action, using provided context
func (e *Engine) Verify(resource ObjectRef, subject ObjectRef, action string, ctx map[string]string) (bool, error) {
	policies, err := e.GetPolicies(resource)
	if err != nil {
		return false, err
	}
	// always ensure subject, action, resource are present in context
	if ctx == nil {
		ctx = map[string]string{}
	}
	ctx["subject"] = subject.ObjectID
	ctx["action"] = action
	ctx["resource"] = resource.String()

	// check each policy for allow
	for _, p := range policies {
		for _, rule := range p.Rules {
			if (rule.Action == "*" || rule.Action == action) && rule.Expr.Eval(ctx) {
				if rule.Action == "*" {
					return true, nil
				}
				// for specific action, require graph relation
				hasRel := e.graph.HasDirectRelation(resource, action, SubjectRef{Object: subject})
				if hasRel {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

const KeyWordCan = "can"

// checkrelationquery parses a string query and checks if a direct relation exists (no policy evaluation)
// expected format: "can <subject> <relation> <resource>"
func (e *Engine) CheckRelationQuery(query string) (bool, error) {
	query = strings.TrimSpace(query)
	parts := strings.Fields(query)
	if len(parts) < 4 || strings.ToLower(parts[0]) != KeyWordCan {
		return false, fmt.Errorf("invalid query format")
	}
	subjectStr := parts[1]
	relation := parts[2]
	resourceStr := parts[3]

	subject, err := parseObjectRef(subjectStr)
	if err != nil {
		return false, fmt.Errorf("invalid subject: %v", err)
	}
	resource, err := parseObjectRef(resourceStr)
	if err != nil {
		return false, fmt.Errorf("invalid resource: %v", err)
	}
	// direct relation check (no policy)
	return e.CheckRelation(resource, relation, SubjectRef{Object: subject}), nil
}

// verifyquery parses a string query and checks access using provided context
// expected format: "can <subject> <action> <resource>"
func (e *Engine) VerifyQuery(query string, ctx map[string]string) (bool, error) {
	query = strings.TrimSpace(query)
	parts := strings.Fields(query)
	if len(parts) < 4 || strings.ToLower(parts[0]) != KeyWordCan {
		return false, fmt.Errorf("invalid query format")
	}
	subjectStr := parts[1]
	action := parts[2]
	resourceStr := parts[3]

	subject, err := parseObjectRef(subjectStr)
	if err != nil {
		return false, fmt.Errorf("invalid subject: %v", err)
	}
	resource, err := parseObjectRef(resourceStr)
	if err != nil {
		return false, fmt.Errorf("invalid resource: %v", err)
	}
	return e.Verify(resource, subject, action, ctx)
}

// newengine creates a new engine instance
func NewEngine(graph *RelationGraph, policyRepo map[string]*Policy) *Engine {
	return &Engine{
		graph:      graph,
		policyRepo: policyRepo,
	}
}
