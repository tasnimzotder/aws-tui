# IAM Service Support — Design

## Goal

Add IAM browsing to the services TUI: list Users, Roles, and Policies (customer-managed) with drill-down into attached policies, group memberships, trust policies, and attached entities.

## Navigation Flow

```
Services → IAM → Sub-menu (Users / Roles / Policies)
                      │
                      ├─ Users list ──→ User sub-menu ──→ Attached Policies
                      │                                ──→ Group Memberships
                      │
                      ├─ Roles list ──→ Role sub-menu ──→ Attached Policies
                      │                                ──→ Trust Policy (text view)
                      │
                      └─ Policies list (customer-managed only)
                              └──→ Attached Entities (users/roles/groups using it)
```

## Users View

Columns: **Name**, **User ID**, **Created**, **Path**

- `ListUsers` API (paginated, client handles all pages)
- Enter drills to a sub-menu: Attached Policies, Group Memberships
- Copy: `c` = username, `C` = ARN

## Roles View

Columns: **Name**, **Description**, **Created**, **Path**

- `ListRoles` API (paginated)
- Enter drills to a sub-menu: Attached Policies, Trust Policy
- Trust Policy shows the `AssumeRolePolicyDocument` as pretty-printed JSON in a text/detail view
- Copy: `c` = role name, `C` = ARN

## Policies View

Columns: **Name**, **Attachment Count**, **Created**, **Updated**

- `ListPolicies` with `Scope: "Local"` (customer-managed only)
- Enter drills to Attached Entities view
- Copy: `c` = policy name, `C` = ARN

## Sub-views

| View | API | Columns |
|------|-----|---------|
| User Attached Policies | `ListAttachedUserPolicies` | Policy Name, ARN |
| User Groups | `ListGroupsForUser` | Group Name, ARN |
| Role Attached Policies | `ListAttachedRolePolicies` | Policy Name, ARN |
| Role Trust Policy | Parsed from `Role.AssumeRolePolicyDocument` | Text view (JSON) |
| Policy Attached Entities | `ListEntitiesForPolicy` | Entity Name, Type (User/Role/Group) |

## New Files

| File | Purpose |
|------|---------|
| `internal/aws/iam/types.go` | IAMUser, IAMRole, IAMPolicy, IAMAttachedPolicy, IAMGroup, IAMPolicyEntity |
| `internal/aws/iam/client.go` | Client with ListUsers, ListRoles, ListPolicies, ListAttachedUserPolicies, ListGroupsForUser, ListAttachedRolePolicies, ListEntitiesForPolicy |
| `internal/aws/iam/client_test.go` | Tests with mock IAM API |
| `internal/tui/services/iam.go` | IAM sub-menu, all list views, trust policy text view |

## Modified Files

| File | Change |
|------|--------|
| `internal/aws/client.go` | Add `IAM *awsiam.Client` to `ServiceClient` |
| `internal/tui/services/root.go` | Add "IAM" to service menu + `handleSelection` |
| `README.md` | Add IAM to supported services |
| `go.mod` | Add `github.com/aws/aws-sdk-go-v2/service/iam` |
