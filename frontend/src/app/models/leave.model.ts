export type LeaveType = 'ANNUAL' | 'SICK' | 'CASUAL' | 'MATERNITY' | 'PATERNITY' | 'UNPAID';
export type LeaveStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'CANCELLED';

export interface LeaveDetails {
  id: number;
  employee_id: number;
  employee_code: string;
  employee_name: string;
  employee_department: string;
  employee_email: string;
  leave_type: LeaveType;
  start_date: string; // YYYY-MM-DD
  end_date: string;   // YYYY-MM-DD
  total_days: number;
  reason: string;
  status: LeaveStatus;
  manager_id?: number | null;
  manager_name?: string | null;
  reviewed_by?: number | null;
  reviewer_name?: string | null;
  review_comments?: string | null;
  reviewed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface PaginatedLeavesResponse {
  data: LeaveDetails[];
  pagination: PaginationMeta;
}

export interface ApplyLeaveRequest {
  employee_id: number;
  leave_type: LeaveType;
  start_date: string;
  end_date: string;
  total_days?: number;
  reason: string;
}

export interface ReviewLeaveRequest {
  manager_id: number;
  comments: string;
}

export interface CancelLeaveRequest {
  employee_id: number;
}

export interface LeaveFilterParams {
  employee_id?: number;
  manager_id?: number;
  status?: LeaveStatus | '';
  leave_type?: LeaveType | '';
  from_date?: string;
  to_date?: string;
  page?: number;
  limit?: number;
}
