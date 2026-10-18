package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"beinex-leave-management/internal/database"
	"beinex-leave-management/internal/models"

	"github.com/jackc/pgx/v5"
)

var (
	ErrLeaveNotFound            = errors.New("leave request not found")
	ErrOverlappingApprovedLeave = errors.New("overlapping approved leave request already exists")
	ErrUnauthorizedManager      = errors.New("only the assigned manager or an administrator can approve or reject this request")
	ErrUnauthorizedEmployee     = errors.New("only the owner of this leave request or an administrator can cancel it")
	ErrCannotApproveCancelled   = errors.New("cannot approve a cancelled leave request")
	ErrCannotApproveRejected    = errors.New("cannot approve a rejected leave request")
	ErrAlreadyApproved          = errors.New("leave request has already been approved")
	ErrNotPending               = errors.New("only pending leave requests can be approved or rejected")
	ErrCannotCancelNonPending   = errors.New("only pending leave requests can be cancelled")
)

// LeaveRepository handles database operations for leaves.
type LeaveRepository struct {
	db *database.DB
}

// NewLeaveRepository creates a new LeaveRepository.
func NewLeaveRepository(db *database.DB) *LeaveRepository {
	return &LeaveRepository{db: db}
}

// CheckOverlappingApprovedLeave checks if an approved leave already covers the date range.
func (r *LeaveRepository) CheckOverlappingApprovedLeave(ctx context.Context, employeeID int64, startDate, endDate time.Time, excludeLeaveID int64) (bool, string, error) {
	query := `
		SELECT
			id,
			TO_CHAR(start_date, 'YYYY-MM-DD') AS start_date_str,
			TO_CHAR(end_date, 'YYYY-MM-DD') AS end_date_str
		FROM leaves
		WHERE employee_id = $1
		  AND status = 'APPROVED'
		  AND id != $2
		  AND start_date <= $4
		  AND end_date >= $3
		LIMIT 1
	`
	var (
		id       int64
		startStr string
		endStr   string
	)
	err := r.db.Pool.QueryRow(ctx, query, employeeID, excludeLeaveID, startDate, endDate).Scan(&id, &startStr, &endStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check overlapping leaves: %w", err)
	}

	overlapMsg := fmt.Sprintf("Leave ID #%d from %s to %s is already approved", id, startStr, endStr)
	return true, overlapMsg, nil
}

// CreateLeave inserts a new leave application record and returns its full details.
func (r *LeaveRepository) CreateLeave(ctx context.Context, leave *models.Leave) (*models.LeaveDetails, error) {
	query := `
		INSERT INTO leaves (
			employee_id,
			leave_type,
			start_date,
			end_date,
			total_days,
			reason,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var newID int64
	err := r.db.Pool.QueryRow(
		ctx,
		query,
		leave.EmployeeID,
		leave.LeaveType,
		leave.StartDate,
		leave.EndDate,
		leave.TotalDays,
		leave.Reason,
		models.LeaveStatusPending,
	).Scan(&newID)

	if err != nil {
		return nil, fmt.Errorf("failed to insert leave record: %w", err)
	}

	return r.GetLeaveDetailsByID(ctx, newID)
}

// GetLeaveDetailsByID retrieves a single leave with full employee and manager metadata.
func (r *LeaveRepository) GetLeaveDetailsByID(ctx context.Context, id int64) (*models.LeaveDetails, error) {
	query := `
		SELECT
			l.id,
			l.employee_id,
			e.employee_code,
			CONCAT(e.first_name, ' ', e.last_name) AS employee_name,
			e.department,
			e.email AS employee_email,
			l.leave_type,
			TO_CHAR(l.start_date, 'YYYY-MM-DD') AS start_date_str,
			TO_CHAR(l.end_date, 'YYYY-MM-DD') AS end_date_str,
			l.total_days,
			l.reason,
			l.status,
			e.manager_id,
			CASE WHEN m.id IS NOT NULL THEN CONCAT(m.first_name, ' ', m.last_name) ELSE NULL END AS manager_name,
			l.reviewed_by,
			CASE WHEN rev.id IS NOT NULL THEN CONCAT(rev.first_name, ' ', rev.last_name) ELSE NULL END AS reviewer_name,
			l.review_comments,
			l.reviewed_at,
			l.created_at,
			l.updated_at
		FROM leaves l
		JOIN employees e ON e.id = l.employee_id
		LEFT JOIN employees m ON m.id = e.manager_id
		LEFT JOIN employees rev ON rev.id = l.reviewed_by
		WHERE l.id = $1
	`

	var item models.LeaveDetails
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.EmployeeID,
		&item.EmployeeCode,
		&item.EmployeeName,
		&item.EmployeeDepartment,
		&item.EmployeeEmail,
		&item.LeaveType,
		&item.StartDate,
		&item.EndDate,
		&item.TotalDays,
		&item.Reason,
		&item.Status,
		&item.ManagerID,
		&item.ManagerName,
		&item.ReviewedBy,
		&item.ReviewerName,
		&item.ReviewComments,
		&item.ReviewedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaveNotFound
		}
		return nil, fmt.Errorf("failed to get leave details: %w", err)
	}

	return &item, nil
}

// ListLeaves queries leaves with dynamic filtering and pagination.
func (r *LeaveRepository) ListLeaves(ctx context.Context, filter models.LeaveFilter) (*models.PaginatedLeavesResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	offset := (filter.Page - 1) * filter.Limit

	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.EmployeeID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("l.employee_id = $%d", argIdx))
		args = append(args, *filter.EmployeeID)
		argIdx++
	}

	if filter.ManagerID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.manager_id = $%d", argIdx))
		args = append(args, *filter.ManagerID)
		argIdx++
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("l.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.LeaveType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("l.leave_type = $%d", argIdx))
		args = append(args, *filter.LeaveType)
		argIdx++
	}

	if filter.FromDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("l.end_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}

	if filter.ToDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("l.start_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// 1. Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM leaves l
		JOIN employees e ON e.id = l.employee_id
		WHERE %s
	`, whereSQL)

	var totalCount int64
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count leaves: %w", err)
	}

	// 2. Fetch paginated records
	selectQuery := fmt.Sprintf(`
		SELECT
			l.id,
			l.employee_id,
			e.employee_code,
			CONCAT(e.first_name, ' ', e.last_name) AS employee_name,
			e.department,
			e.email AS employee_email,
			l.leave_type,
			TO_CHAR(l.start_date, 'YYYY-MM-DD') AS start_date_str,
			TO_CHAR(l.end_date, 'YYYY-MM-DD') AS end_date_str,
			l.total_days,
			l.reason,
			l.status,
			e.manager_id,
			CASE WHEN m.id IS NOT NULL THEN CONCAT(m.first_name, ' ', m.last_name) ELSE NULL END AS manager_name,
			l.reviewed_by,
			CASE WHEN rev.id IS NOT NULL THEN CONCAT(rev.first_name, ' ', rev.last_name) ELSE NULL END AS reviewer_name,
			l.review_comments,
			l.reviewed_at,
			l.created_at,
			l.updated_at
		FROM leaves l
		JOIN employees e ON e.id = l.employee_id
		LEFT JOIN employees m ON m.id = e.manager_id
		LEFT JOIN employees rev ON rev.id = l.reviewed_by
		WHERE %s
		ORDER BY l.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	queryArgs := append(args, filter.Limit, offset)

	rows, err := r.db.Pool.Query(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaves: %w", err)
	}
	defer rows.Close()

	results := make([]models.LeaveDetails, 0)
	for rows.Next() {
		var item models.LeaveDetails
		scanErr := rows.Scan(
			&item.ID,
			&item.EmployeeID,
			&item.EmployeeCode,
			&item.EmployeeName,
			&item.EmployeeDepartment,
			&item.EmployeeEmail,
			&item.LeaveType,
			&item.StartDate,
			&item.EndDate,
			&item.TotalDays,
			&item.Reason,
			&item.Status,
			&item.ManagerID,
			&item.ManagerName,
			&item.ReviewedBy,
			&item.ReviewerName,
			&item.ReviewComments,
			&item.ReviewedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan leave row: %w", scanErr)
		}
		results = append(results, item)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &models.PaginatedLeavesResponse{
		Data: results,
		Pagination: models.PaginationMeta{
			Page:       filter.Page,
			Limit:      filter.Limit,
			Total:      totalCount,
			TotalPages: totalPages,
		},
	}, nil
}

// ApproveLeave performs transactional approval with pessimistic locking and authorization check.
func (r *LeaveRepository) ApproveLeave(ctx context.Context, leaveID int64, managerID int64, comments string) (*models.LeaveDetails, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock the leave row and fetch associated employee and manager details
	lockQuery := `
		SELECT
			l.id,
			l.employee_id,
			l.status,
			l.start_date,
			l.end_date,
			e.manager_id
		FROM leaves l
		JOIN employees e ON e.id = l.employee_id
		WHERE l.id = $1
		FOR UPDATE OF l
	`
	var (
		id             int64
		employeeID     int64
		currentStatus  models.LeaveStatus
		startDate      time.Time
		endDate        time.Time
		empManagerID   *int64
	)
	err = tx.QueryRow(ctx, lockQuery, leaveID).Scan(
		&id,
		&employeeID,
		&currentStatus,
		&startDate,
		&endDate,
		&empManagerID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaveNotFound
		}
		return nil, fmt.Errorf("failed to lock leave: %w", err)
	}

	// 2. Validate current state transitions
	if currentStatus == models.LeaveStatusApproved {
		return nil, ErrAlreadyApproved
	}
	if currentStatus == models.LeaveStatusCancelled {
		return nil, ErrCannotApproveCancelled
	}
	if currentStatus == models.LeaveStatusRejected {
		return nil, ErrCannotApproveRejected
	}
	if currentStatus != models.LeaveStatusPending {
		return nil, ErrNotPending
	}

	// 3. Verify Manager Authorization
	var (
		actingRole     models.EmployeeRole
		actingIsActive bool
	)
	authQuery := `SELECT role, is_active FROM employees WHERE id = $1`
	err = tx.QueryRow(ctx, authQuery, managerID).Scan(&actingRole, &actingIsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorizedManager
		}
		return nil, fmt.Errorf("failed to verify manager: %w", err)
	}
	if !actingIsActive {
		return nil, ErrUnauthorizedManager
	}

	isDirectManager := empManagerID != nil && *empManagerID == managerID
	isAdmin := actingRole == models.RoleAdmin

	if !isDirectManager && !isAdmin {
		return nil, ErrUnauthorizedManager
	}

	// 4. Concurrency check: Ensure no overlapping leave was approved while pending
	overlapCheckQuery := `
		SELECT id
		FROM leaves
		WHERE employee_id = $1
		  AND id != $2
		  AND status = 'APPROVED'
		  AND start_date <= $4
		  AND end_date >= $3
		LIMIT 1
	`
	var conflictingID int64
	err = tx.QueryRow(ctx, overlapCheckQuery, employeeID, leaveID, startDate, endDate).Scan(&conflictingID)
	if err == nil {
		return nil, fmt.Errorf("%w: conflicting approved leave #%d exists for these dates", ErrOverlappingApprovedLeave, conflictingID)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to verify conflict: %w", err)
	}

	// 5. Update leave status to APPROVED
	var reviewCommentPtr *string
	trimmedComments := strings.TrimSpace(comments)
	if trimmedComments != "" {
		reviewCommentPtr = &trimmedComments
	}

	updateQuery := `
		UPDATE leaves
		SET
			status = 'APPROVED',
			reviewed_by = $2,
			review_comments = $3,
			reviewed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateQuery, leaveID, managerID, reviewCommentPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to approve leave: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit approval: %w", err)
	}

	return r.GetLeaveDetailsByID(ctx, leaveID)
}

// RejectLeave performs transactional rejection with authorization check.
func (r *LeaveRepository) RejectLeave(ctx context.Context, leaveID int64, managerID int64, comments string) (*models.LeaveDetails, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock leave row
	lockQuery := `
		SELECT
			l.id,
			l.status,
			e.manager_id
		FROM leaves l
		JOIN employees e ON e.id = l.employee_id
		WHERE l.id = $1
		FOR UPDATE OF l
	`
	var (
		id            int64
		currentStatus models.LeaveStatus
		empManagerID  *int64
	)
	err = tx.QueryRow(ctx, lockQuery, leaveID).Scan(&id, &currentStatus, &empManagerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaveNotFound
		}
		return nil, fmt.Errorf("failed to lock leave: %w", err)
	}

	// 2. Validate status
	if currentStatus == models.LeaveStatusCancelled {
		return nil, errors.New("cannot reject a cancelled leave request")
	}
	if currentStatus == models.LeaveStatusRejected {
		return nil, errors.New("leave request has already been rejected")
	}
	if currentStatus == models.LeaveStatusApproved {
		return nil, errors.New("cannot reject an already approved leave request")
	}
	if currentStatus != models.LeaveStatusPending {
		return nil, ErrNotPending
	}

	// 3. Verify Manager Authorization
	var (
		actingRole     models.EmployeeRole
		actingIsActive bool
	)
	authQuery := `SELECT role, is_active FROM employees WHERE id = $1`
	err = tx.QueryRow(ctx, authQuery, managerID).Scan(&actingRole, &actingIsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorizedManager
		}
		return nil, fmt.Errorf("failed to verify manager: %w", err)
	}
	if !actingIsActive {
		return nil, ErrUnauthorizedManager
	}

	isDirectManager := empManagerID != nil && *empManagerID == managerID
	isAdmin := actingRole == models.RoleAdmin

	if !isDirectManager && !isAdmin {
		return nil, ErrUnauthorizedManager
	}

	// 4. Update status to REJECTED
	var reviewCommentPtr *string
	trimmedComments := strings.TrimSpace(comments)
	if trimmedComments != "" {
		reviewCommentPtr = &trimmedComments
	}

	updateQuery := `
		UPDATE leaves
		SET
			status = 'REJECTED',
			reviewed_by = $2,
			review_comments = $3,
			reviewed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateQuery, leaveID, managerID, reviewCommentPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to reject leave: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit rejection: %w", err)
	}

	return r.GetLeaveDetailsByID(ctx, leaveID)
}

// CancelLeave cancels a pending leave request with ownership verification.
func (r *LeaveRepository) CancelLeave(ctx context.Context, leaveID int64, employeeID int64) (*models.LeaveDetails, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock leave row
	lockQuery := `
		SELECT
			id,
			employee_id,
			status
		FROM leaves
		WHERE id = $1
		FOR UPDATE
	`
	var (
		id            int64
		leaveEmpID    int64
		currentStatus models.LeaveStatus
	)
	err = tx.QueryRow(ctx, lockQuery, leaveID).Scan(&id, &leaveEmpID, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaveNotFound
		}
		return nil, fmt.Errorf("failed to lock leave: %w", err)
	}

	// 2. Validate ownership / admin role
	var (
		actingRole     models.EmployeeRole
		actingIsActive bool
	)
	err = tx.QueryRow(ctx, "SELECT role, is_active FROM employees WHERE id = $1", employeeID).Scan(&actingRole, &actingIsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorizedEmployee
		}
		return nil, fmt.Errorf("failed to verify employee: %w", err)
	}
	if !actingIsActive {
		return nil, ErrUnauthorizedEmployee
	}

	if leaveEmpID != employeeID && actingRole != models.RoleAdmin {
		return nil, ErrUnauthorizedEmployee
	}

	// 3. Validate status
	if currentStatus != models.LeaveStatusPending {
		return nil, fmt.Errorf("%w: current status is %s", ErrCannotCancelNonPending, currentStatus)
	}

	// 4. Update status to CANCELLED
	updateQuery := `
		UPDATE leaves
		SET
			status = 'CANCELLED',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateQuery, leaveID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel leave: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit cancellation: %w", err)
	}

	return r.GetLeaveDetailsByID(ctx, leaveID)
}
