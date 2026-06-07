-- Migration 002: Add enrichment_metadata JSONB column
-- This column stores async scoring results, tags, and quality data from
-- the OpenClaw LLM evaluation pipeline (Phase 2+ of ALMS v2).
-- Default is empty JSON object; nullable on existing rows for zero-downtime.
-- Rollback: ALTER TABLE learnings DROP COLUMN enrichment_metadata;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'learnings' AND column_name = 'enrichment_metadata'
    ) THEN
        ALTER TABLE learnings ADD COLUMN enrichment_metadata JSONB DEFAULT '{}'::jsonb;
    END IF;
END $$;
