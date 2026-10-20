-- 001_init.down.sql
-- Reverse the initial migration

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS feedback;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS registrations;
DROP TABLE IF EXISTS registration_groups;
DROP TABLE IF EXISTS event_custom_fields;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS club_admin_profiles;
DROP TABLE IF EXISTS clubs;
DROP TABLE IF EXISTS student_profiles;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "uuid-ossp";
