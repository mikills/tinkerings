package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	g := NewRelationGraph()

	doc := ObjectRef{Type: "document", ObjectID: "readme"}
	alice := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "alice"},
	}
	folder := ObjectRef{Type: "folder", ObjectID: "drafts"}
	userset := SubjectRef{
		Object:   folder,
		Relation: "owner",
	}

	// write concrete subject
	g.Write(RelationTuple{
		Object:   doc,
		Relation: "viewer",
		Subject:  alice,
	})

	assert.True(t, g.HasDirectRelation(doc, "viewer", alice))

	// write userset subject
	g.Write(RelationTuple{
		Object:   doc,
		Relation: "editor",
		Subject:  userset,
	})

	assert.True(t, g.HasDirectRelation(doc, "editor", userset))
}

func TestDelete(t *testing.T) {
	g := NewRelationGraph()

	doc := ObjectRef{Type: "document", ObjectID: "readme"}
	alice := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "alice"},
	}

	tuple := RelationTuple{
		Object:   doc,
		Relation: "viewer",
		Subject:  alice,
	}

	g.Write(tuple)
	assert.True(t, g.Delete(tuple), "deleting existing tuple should return true")
	assert.False(t, g.HasDirectRelation(doc, "viewer", alice))

	// deleting non-existent tuple
	assert.False(t, g.Delete(tuple), "deleting non-existent tuple should return false")
}

func TestReadTuples(t *testing.T) {
	g := NewRelationGraph()

	doc := ObjectRef{Type: "document", ObjectID: "readme"}
	alice := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "alice"},
	}
	bob := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "bob"},
	}

	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: alice})
	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: bob})
	g.Write(RelationTuple{Object: doc, Relation: "editor", Subject: alice})

	// read specific relation
	viewers := g.ReadTuples(doc, "viewer")
	assert.Len(t, viewers, 2)

	// read all relations
	allTuples := g.ReadTuples(doc, "")
	assert.Len(t, allTuples, 3)
}

func TestHasDirectRelation(t *testing.T) {
	g := NewRelationGraph()

	doc := ObjectRef{Type: "document", ObjectID: "readme"}
	alice := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "alice"},
	}
	bob := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "bob"},
	}
	folder := ObjectRef{Type: "folder", ObjectID: "drafts"}
	userset := SubjectRef{
		Object:   folder,
		Relation: "owner",
	}

	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: alice})
	g.Write(RelationTuple{Object: doc, Relation: "editor", Subject: userset})

	// positive cases
	assert.True(t, g.HasDirectRelation(doc, "viewer", alice))
	assert.True(t, g.HasDirectRelation(doc, "editor", userset))

	// negative cases
	assert.False(t, g.HasDirectRelation(doc, "viewer", bob))
	assert.False(t, g.HasDirectRelation(doc, "editor", alice))
}

func TestHasDeepRelationship(t *testing.T) {
	g := NewRelationGraph()

	doc := ObjectRef{Type: "document", ObjectID: "readme"}
	alice := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "alice"},
	}
	bob := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "bob"},
	}
	folder := ObjectRef{Type: "folder", ObjectID: "drafts"}
	group := ObjectRef{Type: "group", ObjectID: "team"}
	userset := SubjectRef{
		Object:   folder,
		Relation: "owner",
	}
	groupUserset := SubjectRef{
		Object:   group,
		Relation: "member",
	}

	// direct: doc#viewer@alice
	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: alice})
	// userset: doc#editor@folder#owner
	g.Write(RelationTuple{Object: doc, Relation: "editor", Subject: userset})
	// userset: folder#owner@group#member
	g.Write(RelationTuple{Object: folder, Relation: "owner", Subject: groupUserset})
	// direct: group#member@bob
	g.Write(RelationTuple{Object: group, Relation: "member", Subject: bob})

	// alice is direct viewer
	assert.True(t, g.HasDeepRelationship(doc, "viewer", alice))
	// bob is not direct editor, but is via userset chain: doc#editor@folder#owner@group#member@bob
	assert.True(t, g.HasDeepRelationship(doc, "editor", bob))
	// alice is not an editor
	assert.False(t, g.HasDeepRelationship(doc, "editor", alice))
	// bob is not a viewer
	assert.False(t, g.HasDeepRelationship(doc, "viewer", bob))
}

func TestGetSubjects(t *testing.T) {
	g := NewRelationGraph()

	doc := ObjectRef{Type: "document", ObjectID: "readme"}
	alice := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "alice"},
	}
	bob := SubjectRef{
		Object: ObjectRef{Type: "user", ObjectID: "bob"},
	}
	folder := ObjectRef{Type: "folder", ObjectID: "drafts"}
	userset := SubjectRef{
		Object:   folder,
		Relation: "owner",
	}

	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: alice})
	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: bob})
	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: userset})

	subjects := g.GetSubjects(doc, "viewer")

	assert.Len(t, subjects, 3)
	assert.Contains(t, subjects, alice)
	assert.Contains(t, subjects, bob)
	assert.Contains(t, subjects, userset)
}

func TestGetObjects(t *testing.T) {
	g := NewRelationGraph()

	alice := ObjectRef{Type: "user", ObjectID: "alice"}
	aliceSubj := SubjectRef{Object: alice}
	doc1 := ObjectRef{Type: "document", ObjectID: "readme"}
	doc2 := ObjectRef{Type: "document", ObjectID: "guide"}
	doc3 := ObjectRef{Type: "document", ObjectID: "notes"}

	g.Write(RelationTuple{Object: doc1, Relation: "viewer", Subject: aliceSubj})
	g.Write(RelationTuple{Object: doc2, Relation: "viewer", Subject: aliceSubj})
	g.Write(RelationTuple{Object: doc3, Relation: "editor", Subject: aliceSubj})

	viewerObjects := g.GetObjects(alice, "viewer")

	assert.Len(t, viewerObjects, 2)
	assert.Contains(t, viewerObjects, doc1)
	assert.Contains(t, viewerObjects, doc2)
	assert.NotContains(t, viewerObjects, doc3)
}

func TestRealisticDocumentSharingScenario(t *testing.T) {
	// scenario: a document is shared with a group for read access (via group#member userset), and one user is granted write/share directly.
	// verifies: group members can read, only the privileged user can write/share, and others have no access.
	g := NewRelationGraph()

	// uuids for objects and users
	docID := "b7e23ec2-8aaf-4c8e-9b1a-1e2e3c4d5f6a"
	groupID := "a1b2c3d4-e5f6-7a8b-9c0d-ef1234567890"
	userReadID := "11111111-2222-3333-4444-555555555555"
	userWriteShareID := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	userNoAccessID := "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"

	doc := ObjectRef{Type: "document", ObjectID: docID}
	group := ObjectRef{Type: "group", ObjectID: groupID}
	userRead := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: userReadID}}
	userWriteShare := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: userWriteShareID}}
	userNoAccess := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: userNoAccessID}}
	groupUserset := SubjectRef{
		Object:   group,
		Relation: "member",
	}

	// group members get read access to the document via userset
	g.Write(RelationTuple{Object: doc, Relation: "read", Subject: groupUserset})
	// userWriteShare has write and share access directly
	g.Write(RelationTuple{Object: doc, Relation: "write", Subject: userWriteShare})
	g.Write(RelationTuple{Object: doc, Relation: "share", Subject: userWriteShare})
	// group members: userRead and userWriteShare
	g.Write(RelationTuple{Object: group, Relation: "member", Subject: userRead})
	g.Write(RelationTuple{Object: group, Relation: "member", Subject: userWriteShare})

	// check: userRead has read access via group
	assert.True(t, g.HasDeepRelationship(doc, "read", userRead))
	// check: userWriteShare has read access via group
	assert.True(t, g.HasDeepRelationship(doc, "read", userWriteShare))
	// check: userWriteShare has write and share access directly
	assert.True(t, g.HasDeepRelationship(doc, "write", userWriteShare))
	assert.True(t, g.HasDeepRelationship(doc, "share", userWriteShare))
	// check: userRead does not have write or share access
	assert.False(t, g.HasDeepRelationship(doc, "write", userRead))
	assert.False(t, g.HasDeepRelationship(doc, "share", userRead))
	// check: userNoAccess has no access
	assert.False(t, g.HasDeepRelationship(doc, "read", userNoAccess))
	assert.False(t, g.HasDeepRelationship(doc, "write", userNoAccess))
	assert.False(t, g.HasDeepRelationship(doc, "share", userNoAccess))
}

func TestHasDeepRelationship_WithParser(t *testing.T) {
	// this test demonstrates using the parser to create subjectrefs and permissions for the graph
	g := NewRelationGraph()

	// create a document and a service
	doc := ObjectRef{Type: "document", ObjectID: "doc-xyz"}
	serviceStr := "service:svc1#admin->deploy,monitor"
	parsed, err := ParseSubjectPermissions(serviceStr)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// grant deploy and monitor to service admin userset
	for _, perm := range parsed.Actions {
		g.Write(RelationTuple{Object: doc, Relation: perm, Subject: parsed.Subject})
	}

	// add a concrete service admin
	admin := SubjectRef{Object: ObjectRef{Type: "service", ObjectID: "svc1-admin"}}
	g.Write(RelationTuple{Object: parsed.Subject.Object, Relation: parsed.Subject.Relation, Subject: admin})

	// admin should have deploy and monitor via userset
	assert.True(t, g.HasDeepRelationship(doc, "deploy", admin))
	assert.True(t, g.HasDeepRelationship(doc, "monitor", admin))
	// admin should not have unrelated permission
	assert.False(t, g.HasDeepRelationship(doc, "read", admin))
}

func TestRelationGraph_JSONRoundTrip(t *testing.T) {
	g := NewRelationGraph()
	doc := ObjectRef{Type: "document", ObjectID: "doc1"}
	user := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "alice"}}
	group := ObjectRef{Type: "group", ObjectID: "team"}
	userset := SubjectRef{Object: group, Relation: "member"}

	g.Write(RelationTuple{Object: doc, Relation: "read", Subject: user})
	g.Write(RelationTuple{Object: doc, Relation: "edit", Subject: userset})
	g.Write(RelationTuple{Object: group, Relation: "member", Subject: user})

	// marshal to json
	data, err := json.Marshal(g)
	assert.NoError(t, err)

	// unmarshal into a new graph
	g2 := NewRelationGraph()
	assert.NoError(t, json.Unmarshal(data, g2))

	// check that all relationships exist in the new graph
	assert.True(t, g2.HasDirectRelation(doc, "read", user))
	assert.True(t, g2.HasDirectRelation(doc, "edit", userset))
	assert.True(t, g2.HasDirectRelation(group, "member", user))
	// check transitive
	assert.True(t, g2.HasDeepRelationship(doc, "edit", user))
}

func TestRelationGraph_LoadFromJSONString(t *testing.T) {
	jsonStr := `
	[
		{
			"Object": {"Type": "document", "ObjectID": "doc2"},
			"Relation": "read",
			"Subject": {"Object": {"Type": "user", "ObjectID": "bob"}, "Relation": ""}
		},
		{
			"Object": {"Type": "document", "ObjectID": "doc2"},
			"Relation": "edit",
			"Subject": {"Object": {"Type": "group", "ObjectID": "devs"}, "Relation": "member"}
		},
		{
			"Object": {"Type": "group", "ObjectID": "devs"},
			"Relation": "member",
			"Subject": {"Object": {"Type": "user", "ObjectID": "bob"}, "Relation": ""}
		}
	]
	`
	var g RelationGraph
	err := json.Unmarshal([]byte(jsonStr), &g)
	assert.NoError(t, err)

	// parse objects and subjects from string
	docParsed, err := ParseSubjectPermissions("document:doc2->read,edit")
	assert.NoError(t, err)
	bobParsed, err := ParseSubjectPermissions("user:bob->read")
	assert.NoError(t, err)
	groupParsed, err := ParseSubjectPermissions("group:devs->member")
	assert.NoError(t, err)
	userset := SubjectRef{Object: groupParsed.Subject.Object, Relation: "member"}

	// check direct relationship using parsed values
	assert.True(t, g.HasDirectRelation(docParsed.Subject.Object, "read", bobParsed.Subject))
	assert.True(t, g.HasDirectRelation(docParsed.Subject.Object, "edit", userset))
	assert.True(t, g.HasDirectRelation(groupParsed.Subject.Object, "member", bobParsed.Subject))
	// check transitive: bob can edit doc2 via group membership
	assert.True(t, g.HasDeepRelationship(docParsed.Subject.Object, "edit", bobParsed.Subject))
}

func BenchmarkParseSubjectPermissions_UsersetMultiplePermissions(b *testing.B) {
	input := "group:123#member->write,share,delete,archive,comment"
	for i := 0; i < b.N; i++ {
		_, err := ParseSubjectPermissions(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSubjectPermissions_ConcreteSinglePermission(b *testing.B) {
	input := "user:abc->read"
	for i := 0; i < b.N; i++ {
		_, err := ParseSubjectPermissions(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSubjectPermissions_UsersetSinglePermission(b *testing.B) {
	input := "service:svc1#admin->deploy"
	for i := 0; i < b.N; i++ {
		_, err := ParseSubjectPermissions(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSubjectPermissions_Malformed(b *testing.B) {
	input := "group:123#->"
	for i := 0; i < b.N; i++ {
		_, _ = ParseSubjectPermissions(input)
	}
}

func BenchmarkParseSubjectPermissions_Whitespace(b *testing.B) {
	input := "   device:dev42   ->   read , write , update   "
	for i := 0; i < b.N; i++ {
		_, err := ParseSubjectPermissions(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// test valid userset with multiple permissions
func TestParseSubjectPermissions_UsersetMultiplePermissions(t *testing.T) {
	input := "group:123#member->write,share"
	parsed, err := ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "group", ObjectID: "123"}, parsed.Subject.Object)
	assert.Equal(t, "member", parsed.Subject.Relation)
	assert.ElementsMatch(t, []string{"write", "share"}, parsed.Actions)
}

// test valid userset with single permission
func TestParseSubjectPermissions_UsersetSinglePermission(t *testing.T) {
	input := "group:123#member->write"
	parsed, err := ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "group", ObjectID: "123"}, parsed.Subject.Object)
	assert.Equal(t, "member", parsed.Subject.Relation)
	assert.Equal(t, []string{"write"}, parsed.Actions)
}

// test valid concrete subject with single permission
func TestParseSubjectPermissions_ConcreteSinglePermission(t *testing.T) {
	input := "user:abc->read"
	parsed, err := ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "user", ObjectID: "abc"}, parsed.Subject.Object)
	assert.Equal(t, "", parsed.Subject.Relation)
	assert.Equal(t, []string{"read"}, parsed.Actions)
}

// test valid concrete subject with multiple permissions
func TestParseSubjectPermissions_ConcreteMultiplePermissions(t *testing.T) {
	input := "user:abc->read,write"
	parsed, err := ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "user", ObjectID: "abc"}, parsed.Subject.Object)
	assert.Equal(t, "", parsed.Subject.Relation)
	assert.ElementsMatch(t, []string{"read", "write"}, parsed.Actions)
}

// test malformed: missing permissions
func TestParseSubjectPermissions_MissingPermissions(t *testing.T) {
	input := "user:abc#member->"
	_, err := ParseSubjectPermissions(input)
	assert.Error(t, err)
}

// test malformed: missing subject
func TestParseSubjectPermissions_MissingSubject(t *testing.T) {
	input := "->read"
	_, err := ParseSubjectPermissions(input)
	assert.Error(t, err)
}

// test malformed: missing type or object id
func TestParseSubjectPermissions_MissingTypeOrID(t *testing.T) {
	inputs := []string{
		":abc->read",
		"user:->read",
		"user->read",
		"#member->read",
	}
	for _, input := range inputs {
		_, err := ParseSubjectPermissions(input)
		assert.Error(t, err)
	}
}

// test malformed: empty string
func TestParseSubjectPermissions_EmptyString(t *testing.T) {
	_, err := ParseSubjectPermissions("")
	assert.Error(t, err)
}

// test whitespace trimming
func TestParseSubjectPermissions_WhitespaceTrim(t *testing.T) {
	input := " group:123#member -> write , share "
	parsed, err := ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "group", ObjectID: "123"}, parsed.Subject.Object)
	assert.Equal(t, "member", parsed.Subject.Relation)
	assert.ElementsMatch(t, []string{"write", "share"}, parsed.Actions)
}

// test non-user subjects: service and device
func TestParseSubjectPermissions_NonUserSubjects(t *testing.T) {
	// service with userset and multiple permissions
	input := "service:svc1#admin->deploy,monitor"
	parsed, err := ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "service", ObjectID: "svc1"}, parsed.Subject.Object)
	assert.Equal(t, "admin", parsed.Subject.Relation)
	assert.ElementsMatch(t, []string{"deploy", "monitor"}, parsed.Actions)

	// device as concrete subject with single permission
	input = "device:dev42->read"
	parsed, err = ParseSubjectPermissions(input)
	assert.NoError(t, err)
	assert.Equal(t, ObjectRef{Type: "device", ObjectID: "dev42"}, parsed.Subject.Object)
	assert.Equal(t, "", parsed.Subject.Relation)
	assert.Equal(t, []string{"read"}, parsed.Actions)
}
