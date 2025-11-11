1. **Relationship-Based Access (Granular Permissions)**

### **Scenario: Document Collaboration**

- **Setup:**
  - Alice is the owner of a project document.
  - Bob is a viewer.
  - Carol is an editor.

- **Graph Relationships:**
  - `(document:project-plan, owner, user:alice)`
  - `(document:project-plan, viewer, user:bob)`
  - `(document:project-plan, editor, user:carol)`

- **Access Control:**
  - **Alice** can read, edit, delete the document (owner).
  - **Bob** can only read the document (viewer).
  - **Carol** can read and edit, but not delete (editor).

- **How it works:**
  - The system checks the graph to see what relationship the user has to the document and grants permissions accordingly.

---

## 2. **Policy-Based Access (Higher-Level Logic)**

### **Scenario: Company-Wide Security Policy**

- **Setup:**
  - The company has a policy:
    - Only employees in the "Engineering" department can access technical documents.
    - Access is denied from outside the corporate network (e.g., IP not in allowed range).
    - Contractors can only access documents during business hours.

- **Policy Examples:**
  - `allow if department == "Engineering" and ip == "10.0.0.0/8"`
  - `deny if role == "contractor" and not (hour >= 9 and hour <= 17)`

- **Access Control:**
  - **Alice** (engineering, in office) can access technical docs.
  - **Bob** (sales, in office) cannot access technical docs.
  - **Carol** (contractor, tries at 8pm) is denied.

- **How it works:**
  - The system evaluates the policy rules using context attributes (department, IP, role, time) to decide access.

---

## 3. **Combined Usage**

### **Scenario: Secure Document Sharing**

- **Setup:**
  - Alice is an owner of a confidential document.
  - Company policy: Only owners or users in "Legal" can access confidential documents, and only from the office.

- **Graph Relationships:**
  - `(document:confidential, owner, user:alice)`
  - `(document:confidential, viewer, user:bob)`

- **Policy:**
  - `allow if relation == "owner" or department == "Legal"`
  - `deny if ip != "10.0.0.0/8"`

- **Access Control:**
  - **Alice** (owner, in office) can access.
  - **Bob** (viewer, in office, not legal) is denied.
  - **Carol** (legal, remote) is denied due to IP.

- **How it works:**
  - The system checks both the graph (is Alice an owner?) and the policy (is Carol in Legal and in office?) before granting access.

---

## **Summary Table**

| Scenario                        | Relationship-Based Example                | Policy-Based Example                                 |
|----------------------------------|-------------------------------------------|------------------------------------------------------|
| Document Collaboration           | Owner/editor/viewer roles                 |                                                      |
| Company Security Policy          |                                           | Department, IP, role, time-based rules               |
| Secure Document Sharing (Combined)| Owner via graph, Legal via policy         | Policy restricts by department and network location  |

---

**In practice:**
- **Relationship-based** is great for direct, role-based permissions (who can do what).
- **Policy-based** is essential for enforcing business rules, compliance, and dynamic conditions.
