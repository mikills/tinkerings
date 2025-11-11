package main

import (
	"testing"
)

// benchmark for hasdirectrelation with a large number of direct relationships
func BenchmarkHasDirectRelation(b *testing.B) {
	g := NewRelationGraph()
	doc := ObjectRef{Type: "document", ObjectID: "benchdoc"}
	user := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "benchuser"}}

	// add many direct relationships
	for i := 0; i < 10000; i++ {
		g.Write(RelationTuple{
			Object:   doc,
			Relation: "viewer",
			Subject:  SubjectRef{Object: ObjectRef{Type: "user", ObjectID: string(rune(i))}},
		})
	}
	// add the target user
	g.Write(RelationTuple{Object: doc, Relation: "viewer", Subject: user})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !g.HasDirectRelation(doc, "viewer", user) {
			b.Fatal("expected direct relation to exist")
		}
	}
}

// benchmark for hasdeeprelationship with a transitive userset chain
func BenchmarkHasDeepRelationship_Transitive(b *testing.B) {
	g := NewRelationGraph()
	doc := ObjectRef{Type: "document", ObjectID: "benchdoc"}
	group := ObjectRef{Type: "group", ObjectID: "benchgroup"}
	user := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "benchuser"}}
	userset := SubjectRef{Object: group, Relation: "member"}

	// doc#editor@group#member
	g.Write(RelationTuple{Object: doc, Relation: "editor", Subject: userset})
	// group#member@user
	g.Write(RelationTuple{Object: group, Relation: "member", Subject: user})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !g.HasDeepRelationship(doc, "editor", user) {
			b.Fatal("expected transitive relation to exist")
		}
	}
}

// benchmark for hasdeeprelationship with a negative case (no relationship)
func BenchmarkHasDeepRelationship_Negative(b *testing.B) {
	g := NewRelationGraph()
	doc := ObjectRef{Type: "document", ObjectID: "benchdoc"}
	user := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "benchuser"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if g.HasDeepRelationship(doc, "editor", user) {
			b.Fatal("did not expect relation to exist")
		}
	}
}

// benchmark for hasdeeprelationship with a deep userset chain (10 levels)
func BenchmarkHasDeepRelationship_DeepChain(b *testing.B) {
	g := NewRelationGraph()
	depth := 10
	doc := ObjectRef{Type: "document", ObjectID: "benchdoc"}
	user := SubjectRef{Object: ObjectRef{Type: "user", ObjectID: "benchuser"}}

	// build a chain: doc#r0@o1#r1, o1#r1@o2#r2, ..., oN#rN@user
	prevObj := doc
	prevRel := "r0"
	for i := 1; i < depth; i++ {
		nextObj := ObjectRef{Type: "obj", ObjectID: string(rune(i))}
		userset := SubjectRef{Object: nextObj, Relation: "r" + string(rune(i))}
		g.Write(RelationTuple{Object: prevObj, Relation: prevRel, Subject: userset})
		prevObj = nextObj
		prevRel = "r" + string(rune(i))
	}
	// last link to user
	g.Write(RelationTuple{Object: prevObj, Relation: prevRel, Subject: user})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !g.HasDeepRelationship(doc, "r0", user) {
			b.Fatal("expected deep chain relation to exist")
		}
	}
}
