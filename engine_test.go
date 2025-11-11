package main

import (
	"testing"
)

func TestEngine_AddPolicy_Verify(t *testing.T) {
	// create a new relation graph
	graph := NewRelationGraph()

	// create a policy that allows 'read' action for subject "alice"
	policyText := `allow read if subject == "alice"`
	policy := NewPolicy(policyText)
	policyID := "policy1"

	// create a policy repo and add the policy
	policyRepo := map[string]*Policy{
		policyID: policy,
	}

	// create the engine
	engine := NewEngine(graph, policyRepo)

	// create a resource and a subject
	resource := engine.CreateResource("document", "doc123")
	subject := ObjectRef{Type: "user", ObjectID: "alice"}

	// attach the policy to the resource
	err := engine.AddPolicyToResource(resource, policyID)
	if err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	// add 'read' relation in graph for subject
	graph.Write(RelationTuple{
		Object:   resource,
		Relation: "read",
		Subject:  SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "alice"}},
	})

	// verify access for the subject with action "read" (should be allowed)
	ctx := map[string]string{"subject": "alice"}
	allowed, err := engine.Verify(resource, subject, "read", ctx)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !allowed {
		t.Errorf("expected access to be allowed, got denied")
	}

	// verify access for the subject with action "write" (should be denied)
	allowed, err = engine.Verify(resource, subject, "write", ctx)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if allowed {
		t.Errorf("expected access to be denied, got allowed")
	}

	// detach the policy
	err = engine.DeletePolicy(resource, policyID)
	if err != nil {
		t.Fatalf("failed to delete policy: %v", err)
	}

	// verify access after detaching policy (should be denied)
	allowed, err = engine.Verify(resource, subject, "read", ctx)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if allowed {
		t.Errorf("expected access to be denied after policy removal, got allowed")
	}
}

func TestEngineScenarioAPI(t *testing.T) {
	graph := NewRelationGraph()
	engine := NewEngine(graph, map[string]*Policy{})

	t.Run("CreateResource and CreateSubject", func(t *testing.T) {
		doc := engine.CreateResource("document", "doc100")
		userAlice := engine.CreateSubject("user", "alice", "")
		teamLegal := engine.CreateSubject("team", "legal", "member")

		if doc.Type != "document" || doc.ObjectID != "doc100" {
			t.Errorf("resource creation failed")
		}
		if userAlice.Object.Type != "user" || userAlice.Object.ObjectID != "alice" {
			t.Errorf("subject creation failed")
		}
		if teamLegal.Object.Type != "team" || teamLegal.Relation != "member" {
			t.Errorf("userset creation failed")
		}
	})

	t.Run("AddRelation and CheckRelation", func(t *testing.T) {
		doc := ObjectRef{Type: "document", ObjectID: "doc100"}
		alice := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "alice"}}
		bob := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "bob"}}
		teamLegal := SubjectRef{Object: ObjectRef{Type: "team", ObjectID: "legal"}, Relation: "member"}

		engine.AddRelation(doc, "owner", alice)
		engine.AddRelation(doc, "viewer", bob)
		engine.AddRelation(doc, "editor", teamLegal)

		if !engine.CheckRelation(doc, "owner", alice) {
			t.Errorf("owner relation missing")
		}
		if !engine.CheckRelation(doc, "viewer", bob) {
			t.Errorf("viewer relation missing")
		}
		if !engine.CheckRelation(doc, "editor", teamLegal) {
			t.Errorf("editor relation missing")
		}
	})

	t.Run("GetRelation, GetObjects, GetResources", func(t *testing.T) {
		doc := ObjectRef{Type: "document", ObjectID: "doc100"}
		alice := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "alice"}}
		bob := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "bob"}}

		relations := engine.GetRelation(doc, alice)
		if len(relations) != 1 || relations[0] != "owner" {
			t.Errorf("getrelation failed for alice")
		}
		relationsBob := engine.GetRelation(doc, bob)
		if len(relationsBob) != 1 || relationsBob[0] != "viewer" {
			t.Errorf("getrelation failed for bob")
		}

		owners := engine.GetObjects(doc, "owner")
		if len(owners) != 1 || owners[0].ObjectID != "alice" {
			t.Errorf("getobjects failed for owner")
		}
		viewers := engine.GetObjects(doc, "viewer")
		if len(viewers) != 1 || viewers[0].ObjectID != "bob" {
			t.Errorf("getobjects failed for viewer")
		}

		resources := engine.GetResources(ObjectRef{Type: "user", ObjectID: "alice"}, "owner")
		if len(resources) != 1 || resources[0].ObjectID != "doc100" {
			t.Errorf("getresources failed for alice")
		}
	})

	t.Run("AddPolicy, AddPolicyToResource, Validation", func(t *testing.T) {
		doc := ObjectRef{Type: "document", ObjectID: "doc100"}
		policyText := `allow read if subject == "alice"`
		err := engine.AddPolicy("p_read_alice", policyText)
		if err != nil {
			t.Fatalf("addpolicy failed: %v", err)
		}
		// should succeed, resource exists
		err = engine.AddPolicyToResource(doc, "p_read_alice")
		if err != nil {
			t.Fatalf("addpolicytores failed: %v", err)
		}
		// should fail, resource does not exist
		missingDoc := ObjectRef{Type: "document", ObjectID: "doesnotexist"}
		err = engine.AddPolicyToResource(missingDoc, "p_read_alice")
		if err == nil {
			t.Errorf("expected error for non-existent resource, got nil")
		}
	})

	t.Run("RemoveRelation", func(t *testing.T) {
		doc := ObjectRef{Type: "document", ObjectID: "doc100"}
		alice := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "alice"}}
		err := engine.RemoveRelation(doc, "owner", alice)
		if err != nil {
			t.Fatalf("removerelation failed: %v", err)
		}
		if engine.CheckRelation(doc, "owner", alice) {
			t.Errorf("owner relation should be removed")
		}
	})

	t.Run("End-to-End Policy Enforcement", func(t *testing.T) {
		doc := ObjectRef{Type: "document", ObjectID: "doc100"}
		aliceObj := ObjectRef{Type: "user", ObjectID: "alice"}
		bobObj := ObjectRef{Type: "user", ObjectID: "bob"}
		// add read relation for alice
		engine.AddRelation(doc, "read", SubjectRef{Object: aliceObj})
		// verify alice can read
		allowed, err := engine.Verify(doc, aliceObj, "read", map[string]string{"subject": "alice"})
		if err != nil {
			t.Fatalf("verify failed: %v", err)
		}
		if !allowed {
			t.Errorf("expected alice to be allowed to read")
		}
		// verify bob cannot read
		allowed, err = engine.Verify(doc, bobObj, "read", map[string]string{"subject": "bob"})
		if err != nil {
			t.Fatalf("verify failed: %v", err)
		}
		if allowed {
			t.Errorf("expected bob to be denied to read")
		}
	})
	t.Run("FeatureFlag only accessible by CTO department", func(t *testing.T) {
		engine := NewEngine(NewRelationGraph(), map[string]*Policy{})
		// create feature flag resource
		flag := engine.CreateResource("feature_flag", "new-dashboard")
		// create users
		userCTO := engine.CreateSubject("user", "eve", "")
		userEng := engine.CreateSubject("user", "bob", "")
		// add enabled relation for both users
		engine.AddRelation(flag, "enabled", userCTO)
		engine.AddRelation(flag, "enabled", userEng)
		// add policy: only CTO department allowed
		policyText := `allow enabled if department == "DL*"`
		engine.AddPolicy("p_cto_only", policyText)
		engine.AddPolicyToResource(flag, "p_cto_only")
		// check access for CTO user
		allowed, err := engine.Verify(flag, userCTO.Object, "enabled", map[string]string{"department": "CTO"})
		if err != nil {
			t.Fatalf("verify failed: %v", err)
		}
		if !allowed {
			t.Errorf("expected CTO user to be allowed for feature flag")
		}
		// check access for Engineering user
		allowed, err = engine.Verify(flag, userEng.Object, "enabled", map[string]string{"department": "Engineering"})
		if err != nil {
			t.Fatalf("verify failed: %v", err)
		}
		if allowed {
			t.Errorf("expected Engineering user to be denied for feature flag")
		}
	})

	t.Run("AddRelationQuery", func(t *testing.T) {
		engine := NewEngine(NewRelationGraph(), map[string]*Policy{})
		// create a resource
		doc := engine.CreateResource("document", "doc200")
		alice := engine.CreateSubject("user", "alice", "")
		teamLegal := engine.CreateSubject("team", "legal", "member")
		// add relation via query string (new format)
		err := engine.AddRelationQuery("document:doc200 user:alice->read,write team:legal#member->approve")
		if err != nil {
			t.Fatalf("addrelationquery failed: %v", err)
		}
		// verify relations exist
		if !engine.CheckRelation(doc, "read", alice) {
			t.Errorf("read relation missing for alice on doc200")
		}
		if !engine.CheckRelation(doc, "write", alice) {
			t.Errorf("write relation missing for alice on doc200")
		}
		if !engine.CheckRelation(doc, "approve", teamLegal) {
			t.Errorf("approve relation missing for team legal member on doc200")
		}

		// try to add relation for non-existent resource
		err = engine.AddRelationQuery("document:notfound user:alice->read")
		if err == nil {
			t.Errorf("expected error for non-existent resource, got nil")
		}
	})

	t.Run("ListAllResources", func(t *testing.T) {
		engine := NewEngine(NewRelationGraph(), map[string]*Policy{})
		doc1 := engine.CreateResource("document", "docA")
		doc2 := engine.CreateResource("document", "docB")
		user := engine.CreateSubject("user", "alice", "")
		engine.AddRelation(doc1, "owner", user)
		engine.AddRelation(doc2, "viewer", user)
		objects := engine.ListAllResources()
		foundA, foundB := false, false
		for _, obj := range objects {
			if obj.ObjectID == "docA" {
				foundA = true
			}
			if obj.ObjectID == "docB" {
				foundB = true
			}
		}
		if !foundA || !foundB {
			t.Errorf("ListAllObjects did not return all created resources")
		}
	})

	t.Run("CheckRelationQuery", func(t *testing.T) {
		engine := NewEngine(NewRelationGraph(), map[string]*Policy{})
		// create the resource first
		engine.CreateResource("document", "doc300")
		// use AddRelationQuery to add subject and relation
		err := engine.AddRelationQuery("document:doc300 user:alice->read")
		if err != nil {
			t.Fatalf("AddRelationQuery failed: %v", err)
		}
		// should return true for existing relation
		allowed, err := engine.CheckRelationQuery("can user:alice read document:doc300")
		if err != nil {
			t.Fatalf("CheckRelationQuery failed: %v", err)
		}
		if !allowed {
			t.Errorf("expected CheckRelationQuery to return true for user:alice read document:doc300")
		}
		// should return false for non-existent relation
		allowed, err = engine.CheckRelationQuery("can user:alice write document:doc300")
		if err != nil {
			t.Fatalf("CheckRelationQuery failed: %v", err)
		}
		if allowed {
			t.Errorf("expected CheckRelationQuery to return false for user:alice write document:doc300")
		}
	})
}
