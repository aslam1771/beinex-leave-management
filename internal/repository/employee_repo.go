package repository

import (
	"context"
	"errors"
	"fmt"

	"beinex-leave-management/internal/database"
	"beinex-leave-management/internal/models"

	"github.com/jackc/pgx/v5"
)

var (
	ErrEmployeeNotFound = errors.New("employee not found")
	ErrEmployeeInactive = errors.New("employee account is inactive")
)

// EmployeeRepository handles database queries for employees.
type EmployeeRepository struct {
	db *database.DB
}

// NewEmployeeRepository creates a new EmployeeRepository.
func NewEmployeeRepository(db *database.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

// GetByID retrieves an employee record by their ID.
func (r *EmployeeRepository) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	query := `
		SELECT
			id,
			employee_code,
			first_name,
			last_name,
			email,
			role,
			department,
			designation,
			manager_id,
			joining_date,
			is_active,
			created_at,
			updated_at
		FROM employees
		WHERE id = $1
	`
	var emp models.Employee
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&emp.ID,
		&emp.EmployeeCode,
		&emp.FirstName,
		&emp.LastName,
		&emp.Email,
		&emp.Role,
		&emp.Department,
		&emp.Designation,
		&emp.ManagerID,
		&emp.JoiningDate,
		&emp.IsActive,
		&emp.CreatedAt,
		&emp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to query employee: %w", err)
	}
	return &emp, nil
}

// ListAll retrieves all active employees ordered by role and name.
func (r *EmployeeRepository) ListAll(ctx context.Context) ([]models.Employee, error) {
	query := `
		SELECT
			id,
			employee_code,
			first_name,
			last_name,
			email,
			role,
			department,
			designation,
			manager_id,
			joining_date,
			is_active,
			created_at,
			updated_at
		FROM employees
		WHERE is_active = true
		ORDER BY
			CASE role WHEN 'ADMIN' THEN 1 WHEN 'MANAGER' THEN 2 ELSE 3 END,
			first_name ASC
	`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}
	defer rows.Close()

	employees := make([]models.Employee, 0)
	for rows.Next() {
		var emp models.Employee
		err := rows.Scan(
			&emp.ID,
			&emp.EmployeeCode,
			&emp.FirstName,
			&emp.LastName,
			&emp.Email,
			&emp.Role,
			&emp.Department,
			&emp.Designation,
			&emp.ManagerID,
			&emp.JoiningDate,
			&emp.IsActive,
			&emp.CreatedAt,
			&emp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan employee: %w", err)
		}
		employees = append(employees, emp)
	}
	return employees, nil
}

