package models

import (
	"testing"
	"time"
)

func TestLeaveValidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		leave   Leave
		wantErr bool
	}{
		{
			name: "valid leave application",
			leave: Leave{
				LeaveType: LeaveTypeAnnual,
				StartDate: now,
				EndDate:   now.AddDate(0, 0, 3),
				TotalDays: 4.0,
			},
			wantErr: false,
		},
		{
			name: "invalid leave type",
			leave: Leave{
				LeaveType: "INVALID_TYPE",
				StartDate: now,
				EndDate:   now.AddDate(0, 0, 1),
				TotalDays: 2.0,
			},
			wantErr: true,
		},
		{
			name: "end_date before start_date",
			leave: Leave{
				LeaveType: LeaveTypeSick,
				StartDate: now.AddDate(0, 0, 2),
				EndDate:   now,
				TotalDays: 1.0,
			},
			wantErr: true,
		},
		{
			name: "zero or negative days",
			leave: Leave{
				LeaveType: LeaveTypeCasual,
				StartDate: now,
				EndDate:   now,
				TotalDays: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.leave.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Leave.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmployeeRoleIsValid(t *testing.T) {
	if !RoleAdmin.IsValid() {
		t.Errorf("expected RoleAdmin to be valid")
	}
	if !RoleManager.IsValid() {
		t.Errorf("expected RoleManager to be valid")
	}
	if !RoleEmployee.IsValid() {
		t.Errorf("expected RoleEmployee to be valid")
	}
	if EmployeeRole("SUPERADMIN").IsValid() {
		t.Errorf("expected unknown role to be invalid")
	}
}

func TestCalculateLeaveDays(t *testing.T) {
	start := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

	days := CalculateLeaveDays(start, end)
	if days != 5.0 {
		t.Errorf("expected 5.0 days, got %.1f", days)
	}

	sameDay := CalculateLeaveDays(start, start)
	if sameDay != 1.0 {
		t.Errorf("expected 1.0 day for same day leave, got %.1f", sameDay)
	}

	invalidDay := CalculateLeaveDays(end, start)
	if invalidDay != 0 {
		t.Errorf("expected 0 for inverted dates, got %.1f", invalidDay)
	}
}

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-11-10")
	if err != nil {
		t.Fatalf("unexpected error parsing date: %v", err)
	}
	if d.Year() != 2026 || d.Month() != 11 || d.Day() != 10 {
		t.Fatalf("unexpected parsed date: %v", d)
	}

	_, err = ParseDate("invalid-date")
	if err == nil {
		t.Errorf("expected error for invalid date string")
	}

	_, err = ParseDate("")
	if err == nil {
		t.Errorf("expected error for empty date string")
	}
}

