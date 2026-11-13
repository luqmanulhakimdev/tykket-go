-- Migration: 000004_seed_default_users.up.sql

-- 1. Add role column to users table if not exists
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) DEFAULT 'MEMBER';

-- 2. Insert or update the 4 default users with password123 (bcrypt cost 10)
INSERT INTO users (id, email, name, password_hash, auth_provider, avatar_url, role, created_at, updated_at)
VALUES
    (
        'a0000000-0000-0000-0000-000000000001',
        'owner@tykket.dev',
        'Alex Mercer',
        crypt('password123', gen_salt('bf', 10)),
        'LOCAL',
        'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=128&auto=format&fit=crop&q=80',
        'OWNER',
        NOW(),
        NOW()
    ),
    (
        'a0000000-0000-0000-0000-000000000002',
        'admin@tykket.dev',
        'Sarah Connor',
        crypt('password123', gen_salt('bf', 10)),
        'LOCAL',
        'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=128&auto=format&fit=crop&q=80',
        'ADMIN',
        NOW(),
        NOW()
    ),
    (
        'a0000000-0000-0000-0000-000000000003',
        'member@tykket.dev',
        'David Kim',
        crypt('password123', gen_salt('bf', 10)),
        'LOCAL',
        'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=128&auto=format&fit=crop&q=80',
        'MEMBER',
        NOW(),
        NOW()
    ),
    (
        'a0000000-0000-0000-0000-000000000004',
        'viewer@tykket.dev',
        'Emma Watson',
        crypt('password123', gen_salt('bf', 10)),
        'LOCAL',
        'https://images.unsplash.com/photo-1517841905240-472988babdf9?w=128&auto=format&fit=crop&q=80',
        'VIEWER',
        NOW(),
        NOW()
    )
ON CONFLICT (email) DO UPDATE SET
    name = EXCLUDED.name,
    password_hash = EXCLUDED.password_hash,
    avatar_url = EXCLUDED.avatar_url,
    role = EXCLUDED.role,
    updated_at = NOW();

-- 3. Ensure the primary TYK project exists and is owned by Alex Mercer
INSERT INTO projects (id, name, key, description, owner_id, issue_counter, created_at, updated_at)
SELECT 
    'b0000000-0000-0000-0000-000000000001',
    'Tykket Core Platform',
    'TYK',
    'Main workspace for Tykket core frontend & backend development',
    u.id,
    0,
    NOW(),
    NOW()
FROM users u
WHERE u.email = 'owner@tykket.dev'
ON CONFLICT (key) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description;

-- 4. Assign project membership and role in project_members for project TYK
INSERT INTO project_members (id, project_id, user_id, role_id, joined_at)
SELECT
    gen_random_uuid(),
    p.id,
    u.id,
    r.id,
    NOW()
FROM projects p
CROSS JOIN users u
JOIN roles r ON (
    (u.email = 'owner@tykket.dev' AND r.name = 'OWNER') OR
    (u.email = 'admin@tykket.dev' AND r.name = 'ADMIN') OR
    (u.email = 'member@tykket.dev' AND r.name = 'MEMBER') OR
    (u.email = 'viewer@tykket.dev' AND r.name = 'VIEWER')
)
WHERE p.key = 'TYK'
ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    role_id = EXCLUDED.role_id;

