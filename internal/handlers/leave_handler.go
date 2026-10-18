package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"beinex-leave-management/internal/models"
	"beinex-leave-management/internal/repository"
)

// APIError represents the standard error format returned by endpoints.
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// LeaveHandler coordinates HTTP requests for leave management.
type LeaveHandler struct {
	leaveRepo    *repository.LeaveRepository
	employeeRepo *repository.EmployeeRepository
}

// NewLeaveHandler creates a new LeaveHandler.
func NewLeaveHandler(leaveRepo *repository.LeaveRepository, employeeRepo *repository.EmployeeRepository) *LeaveHandler {
	return &LeaveHandler{
		leaveRepo:    leaveRepo,
		employeeRepo: employeeRepo,
	}
}

func (h *LeaveHandler) isReady(w http.ResponseWriter) bool {
	if h == nil || h.leaveRepo == nil || h.employeeRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "Database Unavailable", "Database connection is not initialized")
		return false
	}
	return true
}

// ApplyLeave handles POST /api/leaves
func (h *LeaveHandler) ApplyLeave(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "POST is required")
		return
	}

	var req models.ApplyLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "Request body must be valid JSON: "+err.Error())
		return
	}

	// 1. Basic field validations
	if req.EmployeeID <= 0 {
		writeError(w, http.StatusBadRequest, "Validation Error", "employee_id must be a positive integer")
		return
	}

	leaveType := models.LeaveType(strings.ToUpper(strings.TrimSpace(req.LeaveType)))
	if !leaveType.IsValid() {
		writeError(w, http.StatusBadRequest, "Validation Error", "leave_type must be one of: ANNUAL, SICK, CASUAL, MATERNITY, PATERNITY, UNPAID")
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		writeError(w, http.StatusBadRequest, "Validation Error", "reason is required and cannot be empty")
		return
	}

	// 2. Date parsing and duration calculation
	startDate, err := models.ParseDate(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Validation Error", "invalid start_date: expected YYYY-MM-DD")
		return
	}

	endDate, err := models.ParseDate(req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Validation Error", "invalid end_date: expected YYYY-MM-DD")
		return
	}

	if endDate.Before(startDate) {
		writeError(w, http.StatusBadRequest, "Validation Error", "end_date must be greater than or equal to start_date")
		return
	}

	calculatedDays := models.CalculateLeaveDays(startDate, endDate)
	totalDays := calculatedDays
	if req.TotalDays != nil {
		if *req.TotalDays <= 0 {
			writeError(w, http.StatusBadRequest, "Validation Error", "total_days must be greater than 0")
			return
		}
		if *req.TotalDays > calculatedDays {
			writeError(w, http.StatusBadRequest, "Validation Error", fmt.Sprintf("total_days (%.1f) cannot exceed calendar duration (%.1f)", *req.TotalDays, calculatedDays))
			return
		}
		totalDays = *req.TotalDays
	}

	// 3. Verify Employee existence and status
	emp, err := h.employeeRepo.GetByID(r.Context(), req.EmployeeID)
	if err != nil {
		if errors.Is(err, repository.ErrEmployeeNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Employee with ID %d does not exist", req.EmployeeID))
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Database error checking employee: "+err.Error())
		return
	}

	if !emp.IsActive {
		writeError(w, http.StatusBadRequest, "Inactive Employee", "Inactive employees are not eligible to apply for leave")
		return
	}

	// 4. Overlap validation: check for conflicting approved leaves
	hasOverlap, overlapDetails, err := h.leaveRepo.CheckOverlappingApprovedLeave(r.Context(), req.EmployeeID, startDate, endDate, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Database error checking overlapping leaves: "+err.Error())
		return
	}
	if hasOverlap {
		writeError(w, http.StatusConflict, "Leave Overlap Conflict", "Cannot apply for leave: "+overlapDetails)
		return
	}

	// 5. Create leave application
	leave := &models.Leave{
		EmployeeID: req.EmployeeID,
		LeaveType:  leaveType,
		StartDate:  startDate,
		EndDate:    endDate,
		TotalDays:  totalDays,
		Reason:     reason,
	}

	created, err := h.leaveRepo.CreateLeave(r.Context(), leave)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to create leave application: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// GetEmployeeLeaves handles GET /api/employees/{id}/leaves
func (h *LeaveHandler) GetEmployeeLeaves(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "GET is required")
		return
	}

	empIDStr := r.PathValue("id")
	empID, err := strconv.ParseInt(empIDStr, 10, 64)
	if err != nil || empID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Parameter", "Employee ID must be a positive integer")
		return
	}

	// Verify employee exists
	_, err = h.employeeRepo.GetByID(r.Context(), empID)
	if err != nil {
		if errors.Is(err, repository.ErrEmployeeNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Employee with ID %d does not exist", empID))
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	filter, err := parseFilterParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Query Parameters", err.Error())
		return
	}
	filter.EmployeeID = &empID

	resp, err := h.leaveRepo.ListLeaves(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to retrieve employee leaves: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetManagerPendingLeaves handles GET /api/managers/{id}/pending-leaves
func (h *LeaveHandler) GetManagerPendingLeaves(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "GET is required")
		return
	}

	mgrIDStr := r.PathValue("id")
	mgrID, err := strconv.ParseInt(mgrIDStr, 10, 64)
	if err != nil || mgrID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Parameter", "Manager ID must be a positive integer")
		return
	}

	// Verify manager exists and has permission
	mgr, err := h.employeeRepo.GetByID(r.Context(), mgrID)
	if err != nil {
		if errors.Is(err, repository.ErrEmployeeNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Manager with ID %d does not exist", mgrID))
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	if mgr.Role != models.RoleManager && mgr.Role != models.RoleAdmin {
		writeError(w, http.StatusForbidden, "Forbidden", "Only users with role MANAGER or ADMIN can access manager pending requests")
		return
	}

	filter, err := parseFilterParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Query Parameters", err.Error())
		return
	}

	pendingStatus := models.LeaveStatusPending
	filter.Status = &pendingStatus
	filter.ManagerID = &mgrID

	resp, err := h.leaveRepo.ListLeaves(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to retrieve pending leaves: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ApproveLeave handles POST /api/leaves/{id}/approve and PUT /api/leaves/{id}/approve
func (h *LeaveHandler) ApproveLeave(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "POST or PUT is required")
		return
	}

	leaveID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || leaveID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Parameter", "Leave ID must be a positive integer")
		return
	}

	var req models.ReviewLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "Request body must be valid JSON: "+err.Error())
		return
	}

	if req.ManagerID <= 0 {
		writeError(w, http.StatusBadRequest, "Validation Error", "manager_id is required and must be a positive integer")
		return
	}

	updated, err := h.leaveRepo.ApproveLeave(r.Context(), leaveID, req.ManagerID, req.Comments)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLeaveNotFound):
			writeError(w, http.StatusNotFound, "Not Found", "Leave request not found")
		case errors.Is(err, repository.ErrUnauthorizedManager):
			writeError(w, http.StatusForbidden, "Forbidden", err.Error())
		case errors.Is(err, repository.ErrAlreadyApproved):
			writeError(w, http.StatusConflict, "Conflict", err.Error())
		case errors.Is(err, repository.ErrCannotApproveCancelled),
			errors.Is(err, repository.ErrCannotApproveRejected),
			errors.Is(err, repository.ErrNotPending):
			writeError(w, http.StatusBadRequest, "Invalid Operation", err.Error())
		case errors.Is(err, repository.ErrOverlappingApprovedLeave):
			writeError(w, http.StatusConflict, "Concurrent Conflict", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Internal Server Error", "Approval failed: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// RejectLeave handles POST /api/leaves/{id}/reject and PUT /api/leaves/{id}/reject
func (h *LeaveHandler) RejectLeave(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "POST or PUT is required")
		return
	}

	leaveID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || leaveID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Parameter", "Leave ID must be a positive integer")
		return
	}

	var req models.ReviewLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "Request body must be valid JSON: "+err.Error())
		return
	}

	if req.ManagerID <= 0 {
		writeError(w, http.StatusBadRequest, "Validation Error", "manager_id is required and must be a positive integer")
		return
	}

	updated, err := h.leaveRepo.RejectLeave(r.Context(), leaveID, req.ManagerID, req.Comments)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLeaveNotFound):
			writeError(w, http.StatusNotFound, "Not Found", "Leave request not found")
		case errors.Is(err, repository.ErrUnauthorizedManager):
			writeError(w, http.StatusForbidden, "Forbidden", err.Error())
		case errors.Is(err, repository.ErrNotPending):
			writeError(w, http.StatusBadRequest, "Invalid Operation", err.Error())
		default:
			writeError(w, http.StatusBadRequest, "Invalid Operation", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// CancelLeave handles POST /api/leaves/{id}/cancel and PUT /api/leaves/{id}/cancel
func (h *LeaveHandler) CancelLeave(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "POST or PUT is required")
		return
	}

	leaveID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || leaveID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Parameter", "Leave ID must be a positive integer")
		return
	}

	var req models.CancelLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "Request body must be valid JSON: "+err.Error())
		return
	}

	if req.EmployeeID <= 0 {
		writeError(w, http.StatusBadRequest, "Validation Error", "employee_id is required and must be a positive integer")
		return
	}

	updated, err := h.leaveRepo.CancelLeave(r.Context(), leaveID, req.EmployeeID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrLeaveNotFound):
			writeError(w, http.StatusNotFound, "Not Found", "Leave request not found")
		case errors.Is(err, repository.ErrUnauthorizedEmployee):
			writeError(w, http.StatusForbidden, "Forbidden", err.Error())
		case errors.Is(err, repository.ErrCannotCancelNonPending):
			writeError(w, http.StatusBadRequest, "Invalid Operation", err.Error())
		default:
			writeError(w, http.StatusBadRequest, "Invalid Operation", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// GetLeaveByID handles GET /api/leaves/{id}
func (h *LeaveHandler) GetLeaveByID(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "GET is required")
		return
	}

	leaveID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || leaveID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Parameter", "Leave ID must be a positive integer")
		return
	}

	leave, err := h.leaveRepo.GetLeaveDetailsByID(r.Context(), leaveID)
	if err != nil {
		if errors.Is(err, repository.ErrLeaveNotFound) {
			writeError(w, http.StatusNotFound, "Not Found", "Leave request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, leave)
}

// ListLeaves handles GET /api/leaves with multi-criteria filtering
func (h *LeaveHandler) ListLeaves(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "GET is required")
		return
	}

	filter, err := parseFilterParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid Query Parameters", err.Error())
		return
	}

	// Optional employee_id query param
	if empStr := r.URL.Query().Get("employee_id"); empStr != "" {
		empID, err := strconv.ParseInt(empStr, 10, 64)
		if err != nil || empID <= 0 {
			writeError(w, http.StatusBadRequest, "Invalid Parameter", "employee_id must be a positive integer")
			return
		}
		filter.EmployeeID = &empID
	}

	// Optional manager_id query param
	if mgrStr := r.URL.Query().Get("manager_id"); mgrStr != "" {
		mgrID, err := strconv.ParseInt(mgrStr, 10, 64)
		if err != nil || mgrID <= 0 {
			writeError(w, http.StatusBadRequest, "Invalid Parameter", "manager_id must be a positive integer")
			return
		}
		filter.ManagerID = &mgrID
	}

	resp, err := h.leaveRepo.ListLeaves(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to list leaves: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListEmployees handles GET /api/employees to power the demo employee selector
func (h *LeaveHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	if !h.isReady(w) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "GET is required")
		return
	}

	employees, err := h.employeeRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to retrieve employees: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, employees)
}


// Helper to parse common query parameters (status, leave_type, dates, pagination)
func parseFilterParams(r *http.Request) (models.LeaveFilter, error) {
	q := r.URL.Query()
	filter := models.LeaveFilter{
		Page:  1,
		Limit: 10,
	}

	if pageStr := q.Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return filter, errors.New("page must be an integer >= 1")
		}
		filter.Page = p
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 || l > 100 {
			return filter, errors.New("limit must be an integer between 1 and 100")
		}
		filter.Limit = l
	}

	if statusStr := q.Get("status"); statusStr != "" {
		status := models.LeaveStatus(strings.ToUpper(strings.TrimSpace(statusStr)))
		if !status.IsValid() {
			return filter, fmt.Errorf("invalid status %q: must be PENDING, APPROVED, REJECTED, or CANCELLED", statusStr)
		}
		filter.Status = &status
	}

	if typeStr := q.Get("leave_type"); typeStr != "" {
		lType := models.LeaveType(strings.ToUpper(strings.TrimSpace(typeStr)))
		if !lType.IsValid() {
			return filter, fmt.Errorf("invalid leave_type %q: must be ANNUAL, SICK, CASUAL, MATERNITY, PATERNITY, or UNPAID", typeStr)
		}
		filter.LeaveType = &lType
	}

	if fromStr := q.Get("from_date"); fromStr != "" {
		d, err := models.ParseDate(fromStr)
		if err != nil {
			return filter, errors.New("invalid from_date: expected YYYY-MM-DD")
		}
		filter.FromDate = &d
	}

	if toStr := q.Get("to_date"); toStr != "" {
		d, err := models.ParseDate(toStr)
		if err != nil {
			return filter, errors.New("invalid to_date: expected YYYY-MM-DD")
		}
		filter.ToDate = &d
	}

	return filter, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, errTitle string, message string) {
	writeJSON(w, status, APIError{
		Error:   errTitle,
		Message: message,
		Status:  status,
	})
}
