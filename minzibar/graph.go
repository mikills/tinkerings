package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ObjectRef represents a resource in the system
// Example, type="document", id="123" represents document:123
type ObjectRef struct {
	Type     string
	ObjectID string
}

func (o ObjectRef) String() string {
	return fmt.Sprintf("%s:%s", o.Type, o.ObjectID)
}

// SubjectRef represents a subject performing an action
// Example concrete subject like user:alice or service:api-gateway
// userset like group:admins#member or team:backend#owner
type SubjectRef struct {
	Object   ObjectRef
	Relation string // empty for concrete subjects, set for usersets
}

func (s SubjectRef) String() string {
	if s.Relation == "" {
		return s.Object.String()
	}
	return fmt.Sprintf("%s#%s", s.Object.String(), s.Relation)
}

// RelationTuple is a single relationship: object, relation, subject
type RelationTuple struct {
	Object   ObjectRef
	Relation string
	Subject  SubjectRef
}

func (r RelationTuple) String() string {
	return fmt.Sprintf("(%s, %s, %s)", r.Object, r.Relation, r.Subject)
}

// tupleKey is used for fast lookup of relation tuples
type tupleKey struct {
	object   string
	relation string
	subject  string
}

func makeTupleKey(tuple RelationTuple) tupleKey {
	return tupleKey{
		object:   tuple.Object.String(),
		relation: tuple.Relation,
		subject:  tuple.Subject.String(),
	}
}

// RelationGraph stores and queries relationship tuples
type RelationGraph struct {
	mu sync.RWMutex

	// tuples stores all relation tuples by their unique key
	tuples map[tupleKey]RelationTuple

	// objectIndex maps (object, relation) -> set of subjects
	// allows fast lookup of "who has relation R to object O?"
	objectIndex map[ObjectRef]map[string]map[SubjectRef]struct{}

	// subjectIndex maps (subject, relation) -> set of objects
	// allows fast lookup of "what objects does subject S have relation R to?"
	// only indexes concrete subjects (not usersets)
	subjectIndex map[ObjectRef]map[string]map[ObjectRef]struct{}
}

// ListAllObjects returns all ObjectRef instances referenced in the graph
func (g *RelationGraph) ListAllObjects() []ObjectRef {
	g.mu.RLock()
	defer g.mu.RUnlock()
	objects := make([]ObjectRef, 0, len(g.objectIndex))
	for obj := range g.objectIndex {
		objects = append(objects, obj)
	}
	return objects
}

// NewRelationGraph returns an empty RelationGraph
func NewRelationGraph() *RelationGraph {
	return &RelationGraph{
		tuples:       make(map[tupleKey]RelationTuple),
		objectIndex:  make(map[ObjectRef]map[string]map[SubjectRef]struct{}),
		subjectIndex: make(map[ObjectRef]map[string]map[ObjectRef]struct{}),
	}
}

// Write adds or updates a relation tuple in the graph
func (g *RelationGraph) Write(tuple RelationTuple) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := makeTupleKey(tuple)
	g.tuples[key] = tuple

	// update objectIndex
	if g.objectIndex[tuple.Object] == nil {
		g.objectIndex[tuple.Object] = make(map[string]map[SubjectRef]struct{})
	}
	if g.objectIndex[tuple.Object][tuple.Relation] == nil {
		g.objectIndex[tuple.Object][tuple.Relation] = make(map[SubjectRef]struct{})
	}
	g.objectIndex[tuple.Object][tuple.Relation][tuple.Subject] = struct{}{}

	// update subjectIndex (only for concrete subjects)
	if tuple.Subject.Relation == "" {
		subjectObj := tuple.Subject.Object
		if g.subjectIndex[subjectObj] == nil {
			g.subjectIndex[subjectObj] = make(map[string]map[ObjectRef]struct{})
		}
		if g.subjectIndex[subjectObj][tuple.Relation] == nil {
			g.subjectIndex[subjectObj][tuple.Relation] = make(map[ObjectRef]struct{})
		}
		g.subjectIndex[subjectObj][tuple.Relation][tuple.Object] = struct{}{}
	}
}

// MarshalJSON  implements [JSON MarshalJSON]
func (g *RelationGraph) MarshalJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	tuples := make([]RelationTuple, 0, len(g.tuples))
	for _, t := range g.tuples {
		tuples = append(tuples, t)
	}
	type alias []RelationTuple
	return json.Marshal(alias(tuples))
}

// UnmarshalJSON loads the graph from a JSON array of relation tuples
func (g *RelationGraph) UnmarshalJSON(data []byte) error {
	type alias []RelationTuple
	var tuples alias
	if err := json.Unmarshal(data, &tuples); err != nil {
		return err
	}
	newGraph := NewRelationGraph()
	for _, t := range tuples {
		newGraph.Write(t)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tuples = newGraph.tuples
	g.objectIndex = newGraph.objectIndex
	g.subjectIndex = newGraph.subjectIndex
	return nil
}

// Delete removes a relation tuple from the graph
func (g *RelationGraph) Delete(tuple RelationTuple) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := makeTupleKey(tuple)
	if _, exists := g.tuples[key]; !exists {
		return false
	}

	delete(g.tuples, key)

	// update objectIndex
	if relations := g.objectIndex[tuple.Object]; relations != nil {
		if subjects := relations[tuple.Relation]; subjects != nil {
			delete(subjects, tuple.Subject)
		}
	}

	// update subjectIndex (only for concrete subjects)
	if tuple.Subject.Relation == "" {
		subjectObj := tuple.Subject.Object
		if relations := g.subjectIndex[subjectObj]; relations != nil {
			if objects := relations[tuple.Relation]; objects != nil {
				delete(objects, tuple.Object)
			}
		}
	}

	return true
}

// ReadTuples returns all tuples for an object and relation (or all relations if relation is empty)
func (g *RelationGraph) ReadTuples(object ObjectRef, relation string) []RelationTuple {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []RelationTuple

	if relation == "" {
		// return all relations for this object
		if relations := g.objectIndex[object]; relations != nil {
			for rel, subjects := range relations {
				for subj := range subjects {
					result = append(result, RelationTuple{
						Object:   object,
						Relation: rel,
						Subject:  subj,
					})
				}
			}
		}
	} else {
		// return specific relation
		if relations := g.objectIndex[object]; relations != nil {
			if subjects := relations[relation]; subjects != nil {
				for subj := range subjects {
					result = append(result, RelationTuple{
						Object:   object,
						Relation: relation,
						Subject:  subj,
					})
				}
			}
		}
	}

	return result
}

// HasDirectRelation returns true if the direct tuple (object, relation, subject) exists
func (g *RelationGraph) HasDirectRelation(object ObjectRef, relation string, subject SubjectRef) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := makeTupleKey(RelationTuple{
		Object:   object,
		Relation: relation,
		Subject:  subject,
	})

	_, exists := g.tuples[key]
	return exists
}

// GetSubjects returns all subjects with the given relation to the object
func (g *RelationGraph) GetSubjects(object ObjectRef, relation string) []SubjectRef {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []SubjectRef

	if relations := g.objectIndex[object]; relations != nil {
		if subjects := relations[relation]; subjects != nil {
			for subj := range subjects {
				result = append(result, subj)
			}
		}
	}

	return result
}

// HasDeepRelationship returns true if subject has the relation to object, following userset chains (transitive)
func (g *RelationGraph) HasDeepRelationship(object ObjectRef, relation string, subject SubjectRef) bool {
	var buf [256]byte
	visited := make(map[string]struct{})
	return g.hasDeepRelationshipHelper(object, relation, subject, visited, buf[:0])
}

// hasDeepRelationshipHelper is the recursive helper for HasDeepRelationship
func (g *RelationGraph) hasDeepRelationshipHelper(object ObjectRef, relation string, subject SubjectRef, visited map[string]struct{}, buf []byte) bool {
	// build a unique key for this check to avoid cycles, using the buffer to minimize allocations
	buf = buf[:0]
	buf = append(buf, object.Type...)
	buf = append(buf, ':')
	buf = append(buf, object.ObjectID...)
	buf = append(buf, '|')
	buf = append(buf, relation...)
	buf = append(buf, '|')
	buf = append(buf, subject.Object.Type...)
	buf = append(buf, ':')
	buf = append(buf, subject.Object.ObjectID...)
	if subject.Relation != "" {
		buf = append(buf, '#')
		buf = append(buf, subject.Relation...)
	}
	key := string(buf)

	if _, ok := visited[key]; ok {
		// already visited this triple, avoid infinite loop
		return false
	}
	visited[key] = struct{}{}

	if g.HasDirectRelation(object, relation, subject) {
		return true
	}

	subjects := g.GetSubjects(object, relation)
	for _, subj := range subjects {
		if subj.Relation != "" {
			if g.hasDeepRelationshipHelper(subj.Object, subj.Relation, subject, visited, buf) {
				return true
			}
		}
	}

	return false
}

// GetObjects returns all objects that the subject has the given relation to
func (g *RelationGraph) GetObjects(subject ObjectRef, relation string) []ObjectRef {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []ObjectRef

	if relations := g.subjectIndex[subject]; relations != nil {
		if objects := relations[relation]; objects != nil {
			for obj := range objects {
				result = append(result, obj)
			}
		}
	}

	return result
}

// parsedsubjectpermissions holds the result of parsing a subject and permissions string.
type ParsedSubjectPermissions struct {
	Subject SubjectRef
	Actions []string
}

// parses a string like "group:123#member->write,share" or "user:abc->read".
// returns parsedsubjectpermissions or an error if the input is malformed.
func ParseSubjectPermissions(input string) (ParsedSubjectPermissions, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return ParsedSubjectPermissions{}, errors.New("input string is empty")
	}

	subjectPart, permissionsPart, found := strings.Cut(input, "->")
	if !found {
		return ParsedSubjectPermissions{}, errors.New("missing '->' separator")
	}
	subjectPart = strings.TrimSpace(subjectPart)
	permissionsPart = strings.TrimSpace(permissionsPart)

	if subjectPart == "" || permissionsPart == "" {
		return ParsedSubjectPermissions{}, errors.New("subject or permissions part is empty")
	}

	// parse permissions (comma-separated)
	var permissions []string
	start := 0
	for i := 0; i <= len(permissionsPart); i++ {
		if i == len(permissionsPart) || permissionsPart[i] == ',' {
			perm := strings.TrimSpace(permissionsPart[start:i])
			if perm != "" {
				permissions = append(permissions, perm)
			}
			start = i + 1
		}
	}
	if len(permissions) == 0 {
		return ParsedSubjectPermissions{}, errors.New("no valid permissions found")
	}

	// parse subjectref, supporting both userset and concrete subject
	var subject SubjectRef
	objectStr, relation, hasRelation := strings.Cut(subjectPart, "#")
	objectStr = strings.TrimSpace(objectStr)
	if hasRelation {
		relation = strings.TrimSpace(relation)
		if relation == "" {
			return ParsedSubjectPermissions{}, errors.New("relation is empty in userset")
		}
	}
	object, err := parseObjectRef(objectStr)
	if err != nil {
		return ParsedSubjectPermissions{}, err
	}
	if hasRelation {
		subject = SubjectRef{
			Object:   object,
			Relation: relation,
		}
	} else {
		subject = SubjectRef{
			Object:   object,
			Relation: "",
		}
	}

	return ParsedSubjectPermissions{
		Subject: subject,
		Actions: permissions,
	}, nil
}

// parses a string like "group:123" into an objectref.
// returns an error if the string is not in the form type:id.
func parseObjectRef(input string) (ObjectRef, error) {
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, ":", 2)
	if len(parts) != 2 {
		return ObjectRef{}, errors.New("object must be in the form type:id")
	}
	typ := strings.TrimSpace(parts[0])
	id := strings.TrimSpace(parts[1])
	if typ == "" || id == "" {
		return ObjectRef{}, errors.New("object type or id is empty")
	}
	return ObjectRef{
		Type:     typ,
		ObjectID: id,
	}, nil
}
