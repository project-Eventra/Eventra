-- 002_seed.up.sql
-- Evenrtra Seed Data: Super Admin & Demo Student Only (Clean Initial State)

-- 1. Insert Super Admin User
INSERT INTO users (id, email, password_hash, role, is_active, email_verified)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@evenrtra.edu',
    '$2a$10$xUeNBQMZ.U3DCiASTnl/k.nttZ/8YHhaL3tFge.CakdW5LPeIT.8G', -- Admin@123456
    'super_admin',
    true,
    true
) ON CONFLICT (id) DO NOTHING;

-- 2. Insert Demo Student User & Profile
INSERT INTO users (id, email, password_hash, role, is_active, email_verified)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'student@evenrtra.edu',
    '$2a$10$ou05bBRTJwKomGPIwWFr0ubkrL6YFd2NwKEcgxwFrj/m1xby1GYVe', -- Student@123456
    'student',
    true,
    true
) ON CONFLICT (id) DO NOTHING;

INSERT INTO student_profiles (id, user_id, first_name, last_name, registration_number, department, academic_session, phone, phone_verified)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'Alex',
    'Chen',
    'STU-2026-001',
    'Computer Science & Engineering',
    '2024-2028',
    '+15550192834',
    true
) ON CONFLICT (user_id) DO NOTHING;
