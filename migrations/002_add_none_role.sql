-- =============================================================================
-- Migration 002: Add NONE to user_role enum
-- Allows users registered by merchant to have no role until admin assigns one
-- =============================================================================

ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'NONE';

-- Change default role from 'USER' to 'NONE' for new registrations
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'NONE';
