# Tykket Backend API (Go)

Backend API for **Tykket** (Work Management & Issue Tracking System) built with Go 1.24+ using Clean Architecture (*domain $\rightarrow$ repository $\rightarrow$ usecase $\rightarrow$ delivery*), PostgreSQL (`pgxpool`), and automated database migrations.

---

## 🏗️ Directory Structure (Clean Architecture)

```text
├── cmd/
│   ├── api/main.go            # HTTP Server Entrypoint & Dependency Injection
│   ├── migrate/main.go        # Database Migration CLI runner (golang-migrate)
│   └── seed/main.go           # Database Seeder CLI for demo accounts & tickets
├── internal/
│   ├── config/                # Viper config (.env & environment variable loading)
│   ├── domain/                # Core entities, DTOs, and repository/usecase interfaces
│   ├── repository/postgres/   # PostgreSQL database query implementations (pgxpool)
│   ├── usecase/               # Core business logic (Issue hierarchy, sprint lifecycle, RBAC)
│   └── delivery/http/
│       ├── handler/           # Echo HTTP handlers/controllers
│       ├── middleware/        # JWT Auth, Project RBAC, Structured Logging, Panic Recovery
│       └── router.go          # Route registration & middleware mapping
├── pkg/
│   ├── database/              # PostgreSQL connection pool initializer
│   ├── token/                 # JWT access & refresh token generator and validator
│   ├── response/              # Standard JSON response envelope format (success, data, meta, error)
│   └── validator/             # Request payload struct validator adapter
├── migrations/                # DDL SQL Up & Down migration files
├── docs/                      # OpenAPI 3.0 specification & documentation
├── docker-compose.yml         # PostgreSQL 16 container definition
└── Makefile                   # Command runner (dev, build, test, migrate-up, seed, etc.)
```

---

## 🚀 Getting Started

### 1. Prerequisites
- Go 1.22+ installed
- Docker & Docker Compose

### 2. Environment Configuration
Copy the sample environment file:
```bash
cp .env.example .env
```

### 3. Start PostgreSQL Database
```bash
make docker-up
```

### 4. Run Database Migrations
Apply table schemas, enums, indexes, and initial roles/permissions:
```bash
make migrate-up
```

To roll back migrations:
```bash
make migrate-down
```

### 5. Seed Demo Data (Optional)
Populate sample users, projects, sprints, and hierarchical issues:
```bash
make seed
```
Demo accounts created (`password123`):
- `owner@tykket.dev` (Project Owner)
- `admin@tykket.dev` (Project Admin)
- `member@tykket.dev` (Team Member / Engineer)
- `viewer@tykket.dev` (Stakeholder / Viewer)

### 6. Run API Server
```bash
make dev
```
The server will start at `http://localhost:8080`.
Interactive Swagger UI documentation is available at `http://localhost:8080/docs`.

---

## 🧪 Running Tests & Coverage

Execute the complete test suite across all layers:
```bash
make test
```

To generate and inspect code coverage report (> 90%):
```bash
go test -coverprofile=coverage.out ./internal/... ./pkg/...
go tool cover -func=coverage.out
```

---

## 📡 API Endpoints Overview

### 1. Authentication & Users
| Method | Endpoint | Description | Access |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | Register a new user (email & bcrypt password) | Public |
| `POST` | `/api/v1/auth/login` | Log in and receive JWT token pair | Public |
| `POST` | `/api/v1/auth/refresh` | Renew access token using refresh token | Public |
| `POST` | `/api/v1/auth/google` | Google OAuth authentication | Public |
| `GET`  | `/api/v1/auth/me` | Retrieve profile of the authenticated user | Bearer Token |

### 2. Projects & Workspaces
| Method | Endpoint | Description | Access / Permission |
|---|---|---|---|
| `GET`    | `/api/v1/projects` | List active projects owned or joined by the user | Bearer Token |
| `POST`   | `/api/v1/projects` | Create a new project (auto-assigns creator as Owner) | Bearer Token |
| `GET`    | `/api/v1/projects/:projectId` | Project details and active member list | Bearer Token |
| `PUT`    | `/api/v1/projects/:projectId` | Update project name and description | `project:update` |
| `DELETE` | `/api/v1/projects/:projectId` | Soft delete project (cascades to members, sprints & issues) | `project:delete` |

### 3. Membership & RBAC
| Method | Endpoint | Description | Access / Permission |
|---|---|---|---|
| `GET`    | `/api/v1/projects/:projectId/members` | List active members in a project | Bearer Token |
| `POST`   | `/api/v1/projects/:projectId/members` | Invite/add a member by email | `member:manage` |
| `PUT`    | `/api/v1/projects/:projectId/members/:userId` | Update member role (`ADMIN`, `MEMBER`, `VIEWER`) | `member:manage` |
| `DELETE` | `/api/v1/projects/:projectId/members/:userId` | Soft delete member from project (revokes access) | `member:manage` |

### 4. Sprints & Backlog
| Method | Endpoint | Description | Access / Permission |
|---|---|---|---|
| `GET`    | `/api/v1/projects/:projectId/sprints` | List active sprints within a project | Bearer Token |
| `POST`   | `/api/v1/projects/:projectId/sprints` | Create a new sprint (`PLANNED`) | `sprint:manage` |
| `GET`    | `/api/v1/projects/:projectId/sprints/:sprintId` | Retrieve sprint details | Bearer Token |
| `PUT`    | `/api/v1/projects/:projectId/sprints/:sprintId` | Update sprint metadata | `sprint:manage` |
| `POST`   | `/api/v1/projects/:projectId/sprints/:sprintId/start` | Start sprint (maximum 1 active sprint per project) | `sprint:manage` |
| `POST`   | `/api/v1/projects/:projectId/sprints/:sprintId/complete` | Complete sprint & reconcile incomplete issues | `sprint:manage` |
| `DELETE` | `/api/v1/projects/:projectId/sprints/:sprintId` | Soft delete sprint | `sprint:manage` |
| `POST`   | `/api/v1/projects/:projectId/sprints/bulk-move` | Bulk assign/unassign issues to sprint or backlog | `issue:update` |

### 5. Core Ticket Engine
| Method | Endpoint | Description | Access / Permission |
|---|---|---|---|
| `GET`    | `/api/v1/projects/:projectId/issues` | Filter active issues (backlog, sprint, type, status, pagination) | Bearer Token |
| `POST`   | `/api/v1/projects/:projectId/issues` | Create issue (atomic numbering: `TYK-1`) | `issue:create` |
| `GET`    | `/api/v1/projects/:projectId/issues/:issueId` | Issue details with subtasks & reporter/assignee | Bearer Token |
| `GET`    | `/api/v1/issues/:key` | Direct lookup by global ticket key (e.g., `TYK-42`) | Bearer Token |
| `PUT`    | `/api/v1/projects/:projectId/issues/:issueId` | Update issue details & hierarchy | `issue:update` |
| `PATCH`  | `/api/v1/projects/:projectId/issues/:issueId/move` | Board movement (status & fractional index position) | `issue:update` |
| `DELETE` | `/api/v1/projects/:projectId/issues/:issueId` | Soft delete issue (cascades to subtasks) | `issue:delete` |

### 6. Metrics, Reporting & Data Export
| Method | Endpoint | Description | Access / Permission |
|---|---|---|---|
| `GET` | `/api/v1/projects/:projectId/metrics` | Status and priority distribution metrics | `report:view` |
| `GET` | `/api/v1/projects/:projectId/sprints/:sprintId/burndown` | Daily sprint burndown data | `report:view` |
| `GET` | `/api/v1/projects/:projectId/export/csv` | Stream CSV export of project issues | `report:view` |
| `GET` | `/api/v1/projects/:projectId/export/summary` | Export project summary dump (JSON) | `report:view` |

---

## 🔒 JSON Response Envelope Format

### Success Response:
```json
{
  "success": true,
  "message": "Project created successfully",
  "data": {
    "id": "e5b8a923-...",
    "name": "Tykket Core Platform",
    "key": "TYK"
  }
}
```

### Error Response:
```json
{
  "success": false,
  "message": "validation error: field 'Key' failed validation 'required'",
  "error": "Key is required"
}
```

---

## 🗑️ Soft Delete Architecture & Referential Integrity

All primary entities in Tykket use **Soft Delete** (`deleted_at TIMESTAMPTZ NULL`) instead of hard deletion (`DELETE FROM ...`) to preserve historical data, maintain audit trails, and prevent accidental data loss:

- **Entity Scope:**
  - `projects`
  - `issues`
  - `sprints`
  - `project_members`
  - `users`

- **Transactional Cascading Rules:**
  - **Project Soft-Delete:** Atomically sets `deleted_at = NOW()` on the project and cascades to all its associated `project_members`, `sprints`, and `issues`.
  - **Issue Soft-Delete:** Atomically sets `deleted_at = NOW()` on the parent issue and cascades to all its child subtasks (`parent_id = $1`).
  - **Project Member Removal:** Sets `deleted_at = NOW()` on the membership record, revoking all project RBAC permissions while keeping historical assignee/reporter integrity intact.

- **Index Optimization & Re-invite Compatibility:**
  - Partial unique index `uq_project_user_active` on `project_members(project_id, user_id) WHERE deleted_at IS NULL` allows previously removed team members to be re-invited without conflict.
  - Dedicated B-Tree indexes on `deleted_at` (`idx_projects_deleted_at`, `idx_issues_deleted_at`, `idx_sprints_deleted_at`, `idx_project_members_deleted_at`, `idx_users_deleted_at`) ensure standard queries filtering `WHERE deleted_at IS NULL` maintain maximum execution speed.

---

## 🐳 Containerization & Local Development (Step 10)

Tykket is fully containerized using a multi-stage Docker build producing an ultra-lightweight (~33MB) Alpine production binary.

Run the entire stack (PostgreSQL + Auto-migration + API server) with a single command:
```bash
docker compose up -d --build
```

To seed demo data within the containerized stack:
```bash
docker compose run --rm api /app/seed
```

To run tests in the development container:
```bash
docker compose run --rm api /app/migrate up
```
