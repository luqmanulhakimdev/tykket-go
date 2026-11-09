-- Enable UUID extension (or native gen_random_uuid in Postgres 13+)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 1. ENUMS
-- ============================================================================
CREATE TYPE auth_provider_enum AS ENUM ('LOCAL', 'GOOGLE', 'GITHUB');
CREATE TYPE sprint_status_enum AS ENUM ('PLANNED', 'ACTIVE', 'COMPLETED');
CREATE TYPE issue_type_enum AS ENUM ('EPIC', 'STORY', 'TASK', 'BUG', 'SUBTASK');
CREATE TYPE issue_status_enum AS ENUM ('TODO', 'IN_PROGRESS', 'IN_REVIEW', 'DONE');
CREATE TYPE issue_priority_enum AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'URGENT');

-- ============================================================================
-- 2. IDENTITY & USERS
-- ============================================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    password_hash VARCHAR(255) NULL, -- Nullable if signed in via OAuth
    auth_provider auth_provider_enum NOT NULL DEFAULT 'LOCAL',
    provider_id VARCHAR(255) NULL,
    avatar_url TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- ============================================================================
-- 3. RBAC (GLOBAL DEFINITION)
-- ============================================================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE, -- 'OWNER', 'ADMIN', 'MEMBER', 'VIEWER'
    description TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL UNIQUE, -- Example: 'project:update', 'issue:create', 'sprint:manage'
    description TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ============================================================================
-- 4. PROJECTS & MEMBERSHIP
-- ============================================================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    key VARCHAR(10) NOT NULL UNIQUE, -- Example: 'TYK', ticket key prefix
    description TEXT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    issue_counter BIGINT NOT NULL DEFAULT 0, -- For atomic per-project ticket auto-increment counter
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_projects_key ON projects(key);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);

CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_project_members_user ON project_members(user_id);
CREATE INDEX idx_project_members_project ON project_members(project_id);
CREATE INDEX idx_project_members_deleted_at ON project_members(deleted_at);
CREATE UNIQUE INDEX uq_project_user_active ON project_members(project_id, user_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- 5. SPRINTS
-- ============================================================================
CREATE TABLE sprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    goal TEXT NULL,
    start_date TIMESTAMPTZ NULL,
    end_date TIMESTAMPTZ NULL,
    status sprint_status_enum NOT NULL DEFAULT 'PLANNED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_sprints_project_status ON sprints(project_id, status);
CREATE INDEX idx_sprints_deleted_at ON sprints(deleted_at);

-- ============================================================================
-- 6. ISSUES (CORE TICKET ENGINE)
-- ============================================================================
CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sprint_id UUID NULL REFERENCES sprints(id) ON DELETE SET NULL, -- NULL = in Backlog
    parent_id UUID NULL REFERENCES issues(id) ON DELETE CASCADE,   -- For Epic -> Story or Story -> Subtask
    key VARCHAR(20) NOT NULL UNIQUE,                               -- Example: 'TYK-101'
    issue_number BIGINT NOT NULL,                                  -- Sequential issue number (e.g. 101)
    type issue_type_enum NOT NULL DEFAULT 'TASK',
    status issue_status_enum NOT NULL DEFAULT 'TODO',
    priority issue_priority_enum NOT NULL DEFAULT 'MEDIUM',
    title VARCHAR(255) NOT NULL,
    description TEXT NULL,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assignee_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    story_points INT NULL,
    position DOUBLE PRECISION NOT NULL DEFAULT 65535.0,            -- Fractional index for drag-and-drop board sorting
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

-- Critical indexes for Board & Backlog query performance
CREATE INDEX idx_issues_project_status ON issues(project_id, status);
CREATE INDEX idx_issues_sprint_id ON issues(sprint_id);
CREATE INDEX idx_issues_parent_id ON issues(parent_id);
CREATE INDEX idx_issues_key ON issues(key);
CREATE INDEX idx_issues_position ON issues(position ASC);
CREATE INDEX idx_issues_deleted_at ON issues(deleted_at);

-- ============================================================================
-- 7. INITIAL SEED DATA (ROLES & PERMISSIONS)
-- ============================================================================
INSERT INTO roles (name, description) VALUES
    ('OWNER', 'Full control over the project and settings'),
    ('ADMIN', 'Can manage project members, sprints, and issues'),
    ('MEMBER', 'Can create, edit, and move issues'),
    ('VIEWER', 'Read-only access to board and metrics');

INSERT INTO permissions (code, description) VALUES
    ('project:delete', 'Delete the project'),
    ('project:update', 'Update project configuration'),
    ('member:manage', 'Invite or remove members and modify roles'),
    ('sprint:manage', 'Create, start, and complete sprints'),
    ('issue:create', 'Create issues and subtasks'),
    ('issue:update', 'Update issue details, status, and position'),
    ('issue:delete', 'Delete issues'),
    ('report:view', 'View metrics, burndown, and velocity');

-- Map Permissions to OWNER Role (All Permissions)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'OWNER';

-- Map Permissions to ADMIN Role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'ADMIN' AND p.code NOT IN ('project:delete');

-- Map Permissions to MEMBER Role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'MEMBER' AND p.code IN ('issue:create', 'issue:update', 'report:view');

-- Map Permissions to VIEWER Role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'VIEWER' AND p.code IN ('report:view');

