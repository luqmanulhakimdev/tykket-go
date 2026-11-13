-- Migration: 000004_seed_default_users.down.sql

-- 1. Remove project membership for seeded users in project TYK
DELETE FROM project_members
WHERE user_id IN (
    SELECT id FROM users WHERE email IN (
        'owner@tykket.dev',
        'admin@tykket.dev',
        'member@tykket.dev',
        'viewer@tykket.dev'
    )
);

-- 2. Remove seeded users
DELETE FROM users
WHERE email IN (
    'owner@tykket.dev',
    'admin@tykket.dev',
    'member@tykket.dev',
    'viewer@tykket.dev'
);

-- 3. Drop role column from users table
ALTER TABLE users DROP COLUMN IF EXISTS role;

