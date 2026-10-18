import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Employee } from '../models/employee.model';
import {
  ApplyLeaveRequest,
  CancelLeaveRequest,
  LeaveDetails,
  LeaveFilterParams,
  PaginatedLeavesResponse,
  ReviewLeaveRequest
} from '../models/leave.model';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private readonly baseUrl = 'http://localhost:8080';

  constructor(private http: HttpClient) {}

  // Employees
  getEmployees(): Observable<Employee[]> {
    return this.http.get<Employee[]>(`${this.baseUrl}/api/employees`);
  }

  // System Health
  getHealth(): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/health`);
  }

  // Leaves
  getLeaves(filter?: LeaveFilterParams): Observable<PaginatedLeavesResponse> {
    const params = this.buildHttpParams(filter);
    return this.http.get<PaginatedLeavesResponse>(`${this.baseUrl}/api/leaves`, { params });
  }

  getLeaveById(id: number): Observable<LeaveDetails> {
    return this.http.get<LeaveDetails>(`${this.baseUrl}/api/leaves/${id}`);
  }

  getEmployeeLeaves(employeeId: number, filter?: LeaveFilterParams): Observable<PaginatedLeavesResponse> {
    const params = this.buildHttpParams(filter);
    return this.http.get<PaginatedLeavesResponse>(`${this.baseUrl}/api/employees/${employeeId}/leaves`, { params });
  }

  getManagerPendingLeaves(managerId: number, filter?: LeaveFilterParams): Observable<PaginatedLeavesResponse> {
    const params = this.buildHttpParams(filter);
    return this.http.get<PaginatedLeavesResponse>(`${this.baseUrl}/api/managers/${managerId}/pending-leaves`, { params });
  }

  applyLeave(payload: ApplyLeaveRequest): Observable<LeaveDetails> {
    return this.http.post<LeaveDetails>(`${this.baseUrl}/api/leaves`, payload);
  }

  approveLeave(leaveId: number, payload: ReviewLeaveRequest): Observable<LeaveDetails> {
    return this.http.post<LeaveDetails>(`${this.baseUrl}/api/leaves/${leaveId}/approve`, payload);
  }

  rejectLeave(leaveId: number, payload: ReviewLeaveRequest): Observable<LeaveDetails> {
    return this.http.post<LeaveDetails>(`${this.baseUrl}/api/leaves/${leaveId}/reject`, payload);
  }

  cancelLeave(leaveId: number, payload: CancelLeaveRequest): Observable<LeaveDetails> {
    return this.http.post<LeaveDetails>(`${this.baseUrl}/api/leaves/${leaveId}/cancel`, payload);
  }

  private buildHttpParams(filter?: LeaveFilterParams): HttpParams {
    let params = new HttpParams();
    if (!filter) return params;

    if (filter.employee_id) params = params.set('employee_id', filter.employee_id.toString());
    if (filter.manager_id) params = params.set('manager_id', filter.manager_id.toString());
    if (filter.status) params = params.set('status', filter.status);
    if (filter.leave_type) params = params.set('leave_type', filter.leave_type);
    if (filter.from_date) params = params.set('from_date', filter.from_date);
    if (filter.to_date) params = params.set('to_date', filter.to_date);
    if (filter.page) params = params.set('page', filter.page.toString());
    if (filter.limit) params = params.set('limit', filter.limit.toString());

    return params;
  }
}
