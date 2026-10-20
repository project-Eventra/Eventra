-- 002_seed.up.sql
-- Evenrtra Seed Data: Super Admin, University Clubs, Demo Student, and Campus Events

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

-- 3. Insert University Clubs
INSERT INTO clubs (id, name, description, email, phone, verification_status, is_active, verified_at, verified_by)
VALUES 
(
    '11111111-1111-1111-1111-111111111111',
    'Robotics & AI Society',
    'Advancing student engineering, aerial drones, computer vision, and autonomous rover competitions across universities.',
    'robotics@evenrtra.edu',
    '+15551002001',
    'approved',
    true,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
),
(
    '22222222-2222-2222-2222-222222222222',
    'IEEE Student Branch',
    'The premier technical organization hosting global hackathons, research colloquiums, and hardware workshops.',
    'ieee@evenrtra.edu',
    '+15551002002',
    'approved',
    true,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
),
(
    '33333333-3333-3333-3333-333333333333',
    'Drama & Cultural Guild',
    'Celebrating artistic expression, theatrical productions, musical showcases, and inter-university talent festivals.',
    'drama@evenrtra.edu',
    '+15551002003',
    'approved',
    true,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
),
(
    '44444444-4444-4444-4444-444444444444',
    'Athletics & Sports League',
    'Organizing varsity tournaments, soccer cups, track & field meets, and campus recreation wellness programs.',
    'sports@evenrtra.edu',
    '+15551002004',
    'approved',
    true,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
),
(
    '55555555-5555-5555-5555-555555555555',
    'Debating & Leadership Forum',
    'Cultivating parliamentary debate, public diplomacy, youth parliaments, and leadership summits.',
    'debating@evenrtra.edu',
    '+15551002005',
    'approved',
    true,
    NOW(),
    '00000000-0000-0000-0000-000000000001'
)
ON CONFLICT (id) DO NOTHING;

-- 4. Insert Club Admin Users & Profiles
INSERT INTO users (id, email, password_hash, role, is_active, email_verified)
VALUES 
(
    '10000000-0000-0000-0000-000000000001',
    'robotics@evenrtra.edu',
    '$2a$10$CN/0Sypnm.sdWNC78gMu/OSAjSUKW/5gCgUmVJIksaKfp/VcVh6iy', -- Club@123456
    'club_admin',
    true,
    true
),
(
    '10000000-0000-0000-0000-000000000002',
    'ieee@evenrtra.edu',
    '$2a$10$CN/0Sypnm.sdWNC78gMu/OSAjSUKW/5gCgUmVJIksaKfp/VcVh6iy', -- Club@123456
    'club_admin',
    true,
    true
),
(
    '10000000-0000-0000-0000-000000000003',
    'drama@evenrtra.edu',
    '$2a$10$CN/0Sypnm.sdWNC78gMu/OSAjSUKW/5gCgUmVJIksaKfp/VcVh6iy', -- Club@123456
    'club_admin',
    true,
    true
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO club_admin_profiles (id, user_id, club_id, is_primary)
VALUES 
(
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '11111111-1111-1111-1111-111111111111',
    true
),
(
    '20000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000002',
    '22222222-2222-2222-2222-222222222222',
    true
),
(
    '20000000-0000-0000-0000-000000000003',
    '10000000-0000-0000-0000-000000000003',
    '33333333-3333-3333-3333-333333333333',
    true
)
ON CONFLICT (user_id) DO NOTHING;

-- 5. Insert Live Events
INSERT INTO events (
    id, club_id, name, short_description, description, event_date, registration_deadline,
    venue, participation_type, min_group_size, max_group_size, max_participants,
    registration_fee, has_fee, status
)
VALUES
(
    'a0000000-0000-0000-0000-000000000001',
    '11111111-1111-1111-1111-111111111111',
    'RoboGenesis 2026: Autonomous Drone & Bot Championship',
    'Battle your custom autonomous robots in obstacle navigation and drone racing arenas.',
    'RoboGenesis brings together engineers and robotics teams from across the nation. Features dual arenas: Line Follower Challenge and Autonomous Quadcopter Maze. Cash prizes, certificates, and corporate tech sponsorships.',
    NOW() + INTERVAL '25 days',
    NOW() + INTERVAL '20 days',
    'Engineering Quad & Innovation Arena',
    'group',
    2,
    4,
    80,
    0,
    false,
    'active'
),
(
    'a0000000-0000-0000-0000-000000000002',
    '22222222-2222-2222-2222-222222222222',
    'HackNexus: 48-Hour National AI & Cloud Hackathon',
    'Build transformative AI models and distributed cloud applications over an intense 48-hour sprint.',
    'HackNexus 2026 provides compute clusters, expert industry mentors from leading tech companies, and non-stop coding. Tracks include Generative AI, Sustainable Smart Cities, and Decentralized Security.',
    NOW() + INTERVAL '14 days',
    NOW() + INTERVAL '10 days',
    'Science Complex Center Hall',
    'group',
    1,
    4,
    150,
    0,
    false,
    'active'
),
(
    'a0000000-0000-0000-0000-000000000003',
    '33333333-3333-3333-3333-333333333333',
    'Campus Cultural Fest: Symphony of Lights 2026',
    'A three-day extravaganza of live orchestral bands, theater, art exhibitions, and food carnivals.',
    'Symphony of Lights is our university''s premier annual festival celebrating community, diversity, and artistic brilliance. Includes headline musical performances and open mic stages.',
    NOW() + INTERVAL '30 days',
    NOW() + INTERVAL '28 days',
    'University Grand Amphitheater',
    'individual',
    1,
    1,
    500,
    0,
    false,
    'active'
),
(
    'a0000000-0000-0000-0000-000000000004',
    '44444444-4444-4444-4444-444444444444',
    'Inter-Faculty Futsal Championship 2026',
    'Represent your department in the high-stakes annual five-a-side indoor football tournament.',
    'The official annual campus championship. 16 departmental teams compete for the coveted Chancellor Cup. Trophies, medals, and refreshments provided.',
    NOW() + INTERVAL '18 days',
    NOW() + INTERVAL '12 days',
    'University Indoor Sports Pavilion',
    'group',
    5,
    7,
    112,
    0,
    false,
    'active'
),
(
    'a0000000-0000-0000-0000-000000000005',
    '55555555-5555-5555-5555-555555555555',
    'Presidential Parliamentary Debate Summit',
    'Engage in British Parliamentary debate on international relations, ethics, and economic policy.',
    'Open to all university students. Judged by certified adjudicators. Master your rhetorical delivery, rebuttal strategy, and logical argumentation.',
    NOW() + INTERVAL '35 days',
    NOW() + INTERVAL '32 days',
    'Faculty of Law Moot Court Hall',
    'individual',
    1,
    1,
    64,
    0,
    false,
    'active'
)
ON CONFLICT (id) DO NOTHING;

-- 6. Insert Custom Fields for Events
INSERT INTO event_custom_fields (id, event_id, field_name, field_type, is_required, sort_order)
VALUES 
(
    uuid_generate_v4(),
    'a0000000-0000-0000-0000-000000000002',
    'GitHub or Portfolio URL',
    'text',
    true,
    1
),
(
    uuid_generate_v4(),
    'a0000000-0000-0000-0000-000000000002',
    'T-Shirt Size (S, M, L, XL)',
    'select',
    true,
    2
)
ON CONFLICT DO NOTHING;

-- 7. Insert Demo Registration for student
INSERT INTO registrations (id, event_id, user_id, registration_type, status)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',
    'individual',
    'confirmed'
) ON CONFLICT DO NOTHING;

-- 8. Insert Sample Feedback
INSERT INTO feedback (id, user_id, name, email, category, subject, message, status)
VALUES (
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000002',
    'Alex Chen',
    'student@evenrtra.edu',
    'suggestion',
    'QR Code Scanner at Gate',
    'The digital ticket system is great! Having dedicated tablet scanner stations at the auditorium will streamline queue entry.',
    'reviewed'
) ON CONFLICT DO NOTHING;
