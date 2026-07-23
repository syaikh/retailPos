-- Migration: 011_add_last_login.sql
-- Description: Add last_login timestamp to users table to track when each user last authenticated
-- Created: 2026-06-06

ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login TIMESTAMPTZ;
