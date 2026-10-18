package models

import "time"

// EmployeeRole defines the role of an employee in the system.
type EmployeeRole string

const (
	RoleAdmin    EmployeeRole = "ADMIN"
	RoleManager  EmployeeRole = "MANAGER"
	RoleEmployee EmployeeRole = "EMPLOYEE"
)

// IsValid checks whether the role is one of the valid enum values.
func (r EmployeeRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleEmployee:
		return true
	default:
		return false
	}
}

// Employee represents an employee record in the database.
type Employee struct {
	ID           int64        `json:"id" db:"id"`
	EmployeeCode string       `json:"employee_code" db:"employee_code"`
	FirstName    string       `json:"first_name" db:"first_name"`
	LastName     string       `json:"last_name" db:"last_name"`
	Email        string       `json:"email" db:"email"`
	PasswordHash string       `json:"-" db:"password_hash"` // never exposed in JSON
	Role         EmployeeRole `json:"role" db:"role"`
	Department   string       `json:"department" db:"department"`
	Designation  string       `json:"designation" db:"designation"`
	ManagerID    *int64       `json:"manager_id,omitempty" db:"manager_id"`
	ManagerName  *string      `json:"manager_name,omitempty" db:"manager_name"`
	JoiningDate  time.Time    `json:"joining_date" db:"joining_date"`
	IsActive     bool         `json:"is_active" db:"is_active"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// FullName returns the combined first and last name.
func (e *Employee) FullName() string {
	return e.FirstName + " " + e.LastName
}
