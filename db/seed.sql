-- =============================================================================
-- Beinex Employee Leave Management System
-- Database Seed Data: PostgreSQL
-- File: db/seed.sql
--
-- Safe to rerun idempotently without creating duplicate records.
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. Insert Administrator
-- -----------------------------------------------------------------------------
INSERT INTO employees (
    employee_code,
    first_name,
    last_name,
    email,
    password_hash,
    role,
    department,
    designation,
    manager_id,
    joining_date,
    is_active
)
VALUES (
    'EMP-001',
    'Alex',
    'Taylor',
    'alex.taylor@beinex.com',
    '$2a$10$78K0bVb7O94vA9f848rFke7M1r3fT3eZlO9E2/zE0pU8A5QyqB5r.', -- default placeholder hash
    'ADMIN',
    'Human Resources',
    'HR Administrator',
    NULL,
    '2021-03-01',
    TRUE
)
ON CONFLICT (email) DO UPDATE
SET
    employee_code = EXCLUDED.employee_code,
    first_name    = EXCLUDED.first_name,
    last_name     = EXCLUDED.last_name,
    role          = EXCLUDED.role,
    department    = EXCLUDED.department,
    designation   = EXCLUDED.designation,
    joining_date  = EXCLUDED.joining_date,
    is_active     = EXCLUDED.is_active;

-- -----------------------------------------------------------------------------
-- 2. Insert Engineering Manager
-- -----------------------------------------------------------------------------
INSERT INTO employees (
    employee_code,
    first_name,
    last_name,
    email,
    password_hash,
    role,
    department,
    designation,
    manager_id,
    joining_date,
    is_active
)
VALUES (
    'EMP-002',
    'Sarah',
    'Connor',
    'sarah.connor@beinex.com',
    '$2a$10$78K0bVb7O94vA9f848rFke7M1r3fT3eZlO9E2/zE0pU8A5QyqB5r.',
    'MANAGER',
    'Engineering',
    'Engineering Manager',
    NULL,
    '2021-06-15',
    TRUE
)
ON CONFLICT (email) DO UPDATE
SET
    employee_code = EXCLUDED.employee_code,
    first_name    = EXCLUDED.first_name,
    last_name     = EXCLUDED.last_name,
    role          = EXCLUDED.role,
    department    = EXCLUDED.department,
    designation   = EXCLUDED.designation,
    joining_date  = EXCLUDED.joining_date,
    is_active     = EXCLUDED.is_active;

-- -----------------------------------------------------------------------------
-- 3. Insert Employees reporting to Engineering Manager (Sarah Connor)
-- -----------------------------------------------------------------------------
INSERT INTO employees (
    employee_code,
    first_name,
    last_name,
    email,
    password_hash,
    role,
    department,
    designation,
    manager_id,
    joining_date,
    is_active
)
VALUES
(
    'EMP-003',
    'John',
    'Doe',
    'john.doe@beinex.com',
    '$2a$10$78K0bVb7O94vA9f848rFke7M1r3fT3eZlO9E2/zE0pU8A5QyqB5r.',
    'EMPLOYEE',
    'Engineering',
    'Senior Backend Developer',
    (SELECT id FROM employees WHERE email = 'sarah.connor@beinex.com'),
    '2022-01-10',
    TRUE
),
(
    'EMP-004',
    'Jane',
    'Smith',
    'jane.smith@beinex.com',
    '$2a$10$78K0bVb7O94vA9f848rFke7M1r3fT3eZlO9E2/zE0pU8A5QyqB5r.',
    'EMPLOYEE',
    'Engineering',
    'Frontend Developer',
    (SELECT id FROM employees WHERE email = 'sarah.connor@beinex.com'),
    '2022-04-01',
    TRUE
),
(
    'EMP-005',
    'Michael',
    'Chang',
    'michael.chang@beinex.com',
    '$2a$10$78K0bVb7O94vA9f848rFke7M1r3fT3eZlO9E2/zE0pU8A5QyqB5r.',
    'EMPLOYEE',
    'Engineering',
    'QA Automation Engineer',
    (SELECT id FROM employees WHERE email = 'sarah.connor@beinex.com'),
    '2022-09-20',
    TRUE
)
ON CONFLICT (email) DO UPDATE
SET
    employee_code = EXCLUDED.employee_code,
    first_name    = EXCLUDED.first_name,
    last_name     = EXCLUDED.last_name,
    role          = EXCLUDED.role,
    department    = EXCLUDED.department,
    designation   = EXCLUDED.designation,
    manager_id    = EXCLUDED.manager_id,
    joining_date  = EXCLUDED.joining_date,
    is_active     = EXCLUDED.is_active;

-- -----------------------------------------------------------------------------
-- 4. Sample Leave Requests (Safe to rerun, won't duplicate)
-- -----------------------------------------------------------------------------
INSERT INTO leaves (
    employee_id,
    leave_type,
    start_date,
    end_date,
    total_days,
    reason,
    status,
    reviewed_by,
    review_comments,
    reviewed_at
)
SELECT
    emp.id,
    'ANNUAL',
    '2026-11-10',
    '2026-11-14',
    5.0,
    'Annual family vacation',
    'APPROVED',
    mgr.id,
    'Approved. Enjoy your vacation!',
    '2026-10-15 10:30:00+00'
FROM employees emp
JOIN employees mgr ON mgr.email = 'sarah.connor@beinex.com'
WHERE emp.email = 'john.doe@beinex.com'
  AND NOT EXISTS (
      SELECT 1 FROM leaves l
      WHERE l.employee_id = emp.id
        AND l.start_date = '2026-11-10'
  );

INSERT INTO leaves (
    employee_id,
    leave_type,
    start_date,
    end_date,
    total_days,
    reason,
    status,
    reviewed_by,
    review_comments,
    reviewed_at
)
SELECT
    emp.id,
    'SICK',
    '2026-10-12',
    '2026-10-13',
    2.0,
    'Recovering from seasonal flu',
    'APPROVED',
    mgr.id,
    'Get well soon!',
    '2026-10-12 09:15:00+00'
FROM employees emp
JOIN employees mgr ON mgr.email = 'sarah.connor@beinex.com'
WHERE emp.email = 'jane.smith@beinex.com'
  AND NOT EXISTS (
      SELECT 1 FROM leaves l
      WHERE l.employee_id = emp.id
        AND l.start_date = '2026-10-12'
  );

INSERT INTO leaves (
    employee_id,
    leave_type,
    start_date,
    end_date,
    total_days,
    reason,
    status,
    reviewed_by,
    review_comments,
    reviewed_at
)
SELECT
    emp.id,
    'CASUAL',
    '2026-11-25',
    '2026-11-26',
    2.0,
    'Personal family event',
    'PENDING',
    NULL,
    NULL,
    NULL
FROM employees emp
WHERE emp.email = 'michael.chang@beinex.com'
  AND NOT EXISTS (
      SELECT 1 FROM leaves l
      WHERE l.employee_id = emp.id
        AND l.start_date = '2026-11-25'
  );
