-- PostgreSQL does not directly support removing values from an ENUM.
-- To safely rollback, map any existing IN_QA and READY_FOR_RELEASE issues to IN_REVIEW.
UPDATE issues SET status = 'IN_REVIEW' WHERE status IN ('IN_QA', 'READY_FOR_RELEASE');

