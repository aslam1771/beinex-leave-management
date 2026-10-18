package models

import (
	"errors"
	"time"
)

// LeaveType defines supported leave categories.
type LeaveType string

const (
	LeaveTypeAnnual    LeaveType = "ANNUAL"
	LeaveTypeSick      LeaveType = "SICK"
	LeaveTypeCasual    LeaveType = "CASUAL"
	LeaveTypeMaternity LeaveType = "MATERNITY"
	LeaveTypePaternity LeaveType = "PATERNITY"
	LeaveTypeUnpaid    LeaveType = "UNPAID"
)

// LeaveStatus defines the lifecycle status of a leave application.
type LeaveStatus string

const (
	LeaveStatusPending   LeaveStatus = "PENDING"
	LeaveStatusApproved  LeaveStatus = "APPROVED"
	LeaveStatusRejected  LeaveStatus = "REJECTED"
	LeaveStatusCancelled LeaveStatus = "CANCELLED"
)

// IsValid checks if the leave type is supported.
func (t LeaveType) IsValid() bool {
	switch t {
	case LeaveTypeAnnual, LeaveTypeSick, LeaveTypeCasual, LeaveTypeMaternity, LeaveTypePaternity, LeaveTypeUnpaid:
		return true
	default:
		return false
	}
}

// IsValid checks if the leave status is valid.
func (s LeaveStatus) IsValid() bool {
	switch s {
	case LeaveStatusPending, LeaveStatusApproved, LeaveStatusRejected, LeaveStatusCancelled:
		return true
	default:
		return false
	}
}

// Leave represents a leave request submitted by an employee.
type Leave struct {
	ID             int64       `json:"id" db:"id"`
	EmployeeID     int64       `json:"employee_id" db:"employee_id"`
	EmployeeName   *string     `json:"employee_name,omitempty" db:"employee_name"`
	LeaveType      LeaveType   `json:"leave_type" db:"leave_type"`
	StartDate      time.Time   `json:"start_date" db:"start_date"`
	EndDate        time.Time   `json:"end_date" db:"end_date"`
	TotalDays      float64     `json:"total_days" db:"total_days"`
	Reason         string      `json:"reason" db:"reason"`
	Status         LeaveStatus `json:"status" db:"status"`
	ReviewedBy     *int64      `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewerName   *string     `json:"reviewer_name,omitempty" db:"reviewer_name"`
	ReviewComments *string     `json:"review_comments,omitempty" db:"review_comments"`
	ReviewedAt     *time.Time  `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
}

// Validate validates leave dates and days constraint.
func (l *Leave) Validate() error {
	if !l.LeaveType.IsValid() {
		return errors.New("invalid leave type")
	}
	if l.EndDate.Before(l.StartDate) {
		return errors.New("end_date must be greater than or equal to start_date")
	}
	if l.TotalDays <= 0 {
		return errors.New("total_days must be greater than 0")
	}
	return nil
}

// CalculateLeaveDays calculates inclusive days between start and end date.
func CalculateLeaveDays(start, end time.Time) float64 {
	s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	e := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if e.Before(s) {
		return 0
	}
	return (e.Sub(s).Hours() / 24.0) + 1.0
}

// ParseDate parses date strings formatted as YYYY-MM-DD or RFC3339.
func ParseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("date is required")
	}
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, errors.New("invalid date format: expected YYYY-MM-DD")
}

// ApplyLeaveRequest defines the payload for creating a leave request.
type ApplyLeaveRequest struct {
	EmployeeID int64    `json:"employee_id"`
	LeaveType  string   `json:"leave_type"`
	StartDate  string   `json:"start_date"` // YYYY-MM-DD
	EndDate    string   `json:"end_date"`   // YYYY-MM-DD
	TotalDays  *float64 `json:"total_days,omitempty"`
	Reason     string   `json:"reason"`
}

// ReviewLeaveRequest defines the payload for approving or rejecting a leave request.
type ReviewLeaveRequest struct {
	ManagerID int64  `json:"manager_id"`
	Comments  string `json:"comments"`
}

// CancelLeaveRequest defines the payload for cancelling a pending leave request.
type CancelLeaveRequest struct {
	EmployeeID int64 `json:"employee_id"`
}

// LeaveDetails provides enriched leave data with employee and manager names.
type LeaveDetails struct {
	ID                 int64       `json:"id"`
	EmployeeID         int64       `json:"employee_id"`
	EmployeeCode       string      `json:"employee_code"`
	EmployeeName       string      `json:"employee_name"`
	EmployeeDepartment string      `json:"employee_department"`
	EmployeeEmail      string      `json:"employee_email"`
	LeaveType          LeaveType   `json:"leave_type"`
	StartDate          string      `json:"start_date"` // YYYY-MM-DD
	EndDate            string      `json:"end_date"`   // YYYY-MM-DD
	TotalDays          float64     `json:"total_days"`
	Reason             string      `json:"reason"`
	Status             LeaveStatus `json:"status"`
	ManagerID          *int64      `json:"manager_id,omitempty"`
	ManagerName        *string     `json:"manager_name,omitempty"`
	ReviewedBy         *int64      `json:"reviewed_by,omitempty"`
	ReviewerName       *string     `json:"reviewer_name,omitempty"`
	ReviewComments     *string     `json:"review_comments,omitempty"`
	ReviewedAt         *time.Time  `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// LeaveFilter holds criteria for searching and paginating leaves.
type LeaveFilter struct {
	EmployeeID *int64
	ManagerID  *int64
	Status     *LeaveStatus
	LeaveType  *LeaveType
	FromDate   *time.Time
	ToDate     *time.Time
	Page       int
	Limit      int
}

// PaginationMeta contains pagination details for responses.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedLeavesResponse wraps paginated leave results.
type PaginatedLeavesResponse struct {
	Data       []LeaveDetails `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

