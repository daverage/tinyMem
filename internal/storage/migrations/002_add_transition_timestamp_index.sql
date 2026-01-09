-- tinyMem v5.3 Performance Optimization
-- Migration: 002
-- Description: Add timestamp index to ledger_state_transitions for chronological queries
-- Idempotent: Yes (uses IF NOT EXISTS)

-- ============================================================================
-- PERFORMANCE: Add index for timestamp-based queries
-- ============================================================================
-- Many diagnostic and audit operations query transitions by timestamp
-- This index significantly improves query performance for chronological access

CREATE INDEX IF NOT EXISTS idx_transitions_timestamp ON ledger_state_transitions(timestamp);
