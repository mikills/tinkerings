# Minzibar Policy Engine

## Overview

Minzibar is a flexible access control system that combines **relationship graphs** and **policy rules** to provide robust, context-aware authorization for resources like documents, files, or other entities.

This README explains how relations and policies work together in the engine, using a real-life scenario.

---

## How Relations and Policies Work Together

### 1. Relationship Graph (Granular Permissions)

The **relationship graph** encodes direct permissions and organizational structure using tuples:

```
(object, relation, subject)
```

For example:
- `(document:confidential-report, owner, user:alice)` — Alice is the owner of the confidential report.
- `(document:confidential-report, viewer, user:bob)` — Bob is a viewer.
- Carol is a contractor (no direct relation to the document).

### 2. Policy Attachment (Business Logic)

**Policies** are attached to resources via the graph and encode business rules:

Examples:
- `allow * if department == "Legal"`
- `deny delete if role == "contractor"`
- `allow read if relation == "owner"`

### 3. Access Request and Evaluation

When a user tries to access a resource, the engine performs the following steps:

#### Example Scenario

**Actors & Resources**
- Alice: Employee in Legal
- Bob: Employee in Engineering
- Carol: Contractor
- Document: `document:confidential-report`

**Access Request**
- Alice tries to **delete** the confidential report.

**Context Provided**
```go
ctx := {
  "subject": "alice",
  "department": "Legal",
  "role": "employee"
}
```

**Engine Workflow**
1. **Find Policies** — The engine queries the graph for all policies attached to the document.
2. **Evaluate Policies** — For each policy:
   - If `allow * if department == "Legal"` matches, Alice is allowed any action.
   - If `deny delete if role == "contractor"` matches, access would be denied (not applicable to Alice).
   - If `allow read if relation == "owner"` matches, Alice would be allowed to read if she is the owner (checked via the graph).
3. **Check Relations** — For policies referencing relations, the engine checks the graph for the required relationship.
4. **Final Decision** — Alice matches the `allow * if department == "Legal"` policy, so she is allowed to delete the document.

#### Another Example

Carol (contractor) tries to **delete** the document.

**Context**
```go
ctx := {
  "subject": "carol",
  "department": "Legal",
  "role": "contractor"
}
```
- The engine finds the `deny delete if role == "contractor"` policy and denies access, regardless of other policies.

---

## Summary

- **Relations** define who is connected to what (e.g., Alice is an owner).
- **Policies** define business rules for access (e.g., only Legal can do anything, contractors cannot delete).
- The **engine**:
  - Uses the graph to discover relationships and attached policies.
  - Evaluates policies using both context and relationships.
  - Makes a decision based on both granular permissions and high-level business logic.

This combination enables fine-grained, flexible, and context-aware access control for any resource in your system.

---

## How to Create a Relation with a Query

You can create relations between resources and subjects (users, groups, teams) using a simple query string format.

### **Query Format**

```
<resource_type>:<resource_id> <subject_type>:<subject_id>[#<relation>] -> <action1>[,<action2>,...]
```

### **Examples**

- **Concrete user:**
  ```
  feature_flag:new-dashboard user:user1->enabled
  ```
  This creates a relation tuple:
  ```
  (feature_flag:new-dashboard, enabled, user:user1)
  ```
  Meaning: user `user1` is enabled for the feature flag.

- **Userset (group/team):**
  ```
  feature_flag:new-dashboard team:legal#member->enabled
  ```
  This creates a relation tuple:
  ```
  (feature_flag:new-dashboard, enabled, team:legal#member)
  ```
  Meaning: all members of the legal team are enabled for the feature flag.

- **Multiple actions:**
  ```
  document:doc123 user:alice->read,write
  ```
  This creates two relation tuples:
  ```
  (document:doc123, read, user:alice)
  (document:doc123, write, user:alice)
  ```

### **How It Works**

- The resource is specified as `<resource_type>:<resource_id>`.
- The subject can be a user (`user:user1`) or a userset (`team:legal#member`).
- The actions (relations) are listed after the `->` and can be comma-separated.
- The system parses the query and adds the corresponding relation tuples to the graph.

### **Note on Usersets**

- The part after `#` (e.g., `member`) specifies the relation within the group or team.
- This enables role-based and transitive access control.

---

**This query format makes it easy to create and manage permissions for users, groups, and resources in one step.**