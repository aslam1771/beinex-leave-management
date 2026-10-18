-- =============================================================================
-- Beinex Employee Leave Management System
-- Database Schema: PostgreSQL
-- File: db/schema.sql
-- =============================================================================

-- Ensure updated_at trigger function exists
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -----------------------------------------------------------------------------
-- Table: employees
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS employees (
    id BIGSERIAL PRIMARY KEY,
    employee_code VARCHAR(30) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(30) NOT NULL DEFAULT 'EMPLOYEE' CHECK (role IN ('ADMIN', 'MANAGER', 'EMPLOYEE')),
    department VARCHAR(100) NOT NULL,
    designation VARCHAR(100) NOT NULL,
    manager_id BIGINT REFERENCES employees(id) ON DELETE SET NULL,
    joining_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for employees
CREATE INDEX IF NOT EXISTS idx_employees_email ON employees(email);
CREATE INDEX IF NOT EXISTS idx_employees_employee_code ON employees(employee_code);
CREATE INDEX IF NOT EXISTS idx_employees_manager_id ON employees(manager_id);
CREATE INDEX IF NOT EXISTS idx_employees_role ON employees(role);
CREATE INDEX IF NOT EXISTS idx_employees_department ON employees(department);
CREATE INDEX IF NOT EXISTS idx_employees_active ON employees(is_active);

-- Trigger for employees updated_at
DROP TRIGGER IF EXISTS trg_employees_updated_at ON employees;
CREATE TRIGGER trg_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- -----------------------------------------------------------------------------
-- Table: leaves
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS leaves (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    leave_type VARCHAR(30) NOT NULL CHECK (leave_type IN ('ANNUAL', 'SICK', 'CASUAL', 'MATERNITY', 'PATERNITY', 'UNPAID')),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_days NUMERIC(4, 1) NOT NULL CHECK (total_days > 0),
    reason TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'CANCELLED')),
    reviewed_by BIGINT REFERENCES employees(id) ON DELETE SET NULL,
    review_comments TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_leave_dates CHECK (end_date >= start_date)
);

-- Indexes for leaves
CREATE INDEX IF NOT EXISTS idx_leaves_employee_id ON leaves(employee_id);
CREATE INDEX IF NOT EXISTS idx_leaves_status ON leaves(status);
CREATE INDEX IF NOT EXISTS idx_leaves_dates ON leaves(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_leaves_reviewed_by ON leaves(reviewed_by);
CREATE INDEX IF NOT EXISTS idx_leaves_type ON leaves(leave_type);
CREATE INDEX IF NOT EXISTS idx_leaves_created_at ON leaves(created_at DESC);

-- Trigger for leaves updated_at
DROP TRIGGER IF EXISTS trg_leaves_updated_at ON leaves;
CREATE TRIGGER trg_leaves_updated_at
    BEFORE UPDATE ON leaves
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
