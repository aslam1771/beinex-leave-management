import { Component, OnInit, effect, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { LeaveDetails, LeaveType, PaginationMeta } from '../../models/leave.model';

@Component({
  selector: 'app-manager-approvals',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './manager-approvals.component.html',
  styleUrls: ['./manager-approvals.component.css']
})
export class ManagerApprovalsComponent implements OnInit {
  readonly loading = signal(true);
  readonly processingId = signal<number | null>(null);
  readonly pendingLeaves = signal<LeaveDetails[]>([]);
  readonly pagination = signal<PaginationMeta>({ page: 1, limit: 10, total: 0, total_pages: 1 });

  // Filters
  typeFilter = '';
  fromDate = '';
  toDate = '';
  currentPage = 1;
  pageSize = 10;

  // Comments keyed by leave ID
  comments: { [key: number]: string } = {};

  // Detail modal
  selectedLeave = signal<LeaveDetails | null>(null);

  constructor(
    public auth: AuthService,
    private api: ApiService,
    private notify: NotificationService
  ) {
    // Reload when active user changes
    effect(() => {
      const user = this.auth.currentUser();
      if (user) {
        this.currentPage = 1;
        this.fetchPendingApprovals();
      }
    });
  }

  ngOnInit(): void {
    this.fetchPendingApprovals();
  }

  fetchPendingApprovals(): void {
    const user = this.auth.currentUser();
    if (!user) return;

    if (!this.auth.isManager()) {
      this.loading.set(false);
      this.pendingLeaves.set([]);
      return;
    }

    this.loading.set(true);

    const filter = {
      leave_type: (this.typeFilter || undefined) as LeaveType,
      from_date: this.fromDate || undefined,
      to_date: this.toDate || undefined,
      page: this.currentPage,
      limit: this.pageSize
    };

    this.api.getManagerPendingLeaves(user.id, filter).subscribe({
      next: (res) => {
        this.pendingLeaves.set(res.data || []);
        if (res.pagination) {
          this.pagination.set(res.pagination);
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.notify.error('Error', 'Failed to load manager approvals: ' + (err.error?.message || err.message));
      }
    });
  }

  approve(leave: LeaveDetails): void {
    const user = this.auth.currentUser();
    if (!user) return;

    const comment = this.comments[leave.id] || '';
    this.processingId.set(leave.id);

    this.api.approveLeave(leave.id, { manager_id: user.id, comments: comment }).subscribe({
      next: () => {
        this.processingId.set(null);
        this.notify.success(
          'Approved!',
          `Leave request #${leave.id} for ${leave.employee_name} has been approved.`
        );
        delete this.comments[leave.id];
        this.fetchPendingApprovals();
      },
      error: (err) => {
        this.processingId.set(null);
        this.notify.error('Approval Failed', err.error?.message || 'Failed to approve leave request.');
      }
    });
  }

  reject(leave: LeaveDetails): void {
    const user = this.auth.currentUser();
    if (!user) return;

    const comment = this.comments[leave.id] || '';
    if (!comment.trim()) {
      alert('Please provide a comment explaining the reason for rejection.');
      return;
    }

    if (!confirm(`Are you sure you want to reject the leave request for ${leave.employee_name}?`)) {
      return;
    }

    this.processingId.set(leave.id);

    this.api.rejectLeave(leave.id, { manager_id: user.id, comments: comment }).subscribe({
      next: () => {
        this.processingId.set(null);
        this.notify.success(
          'Rejected',
          `Leave request #${leave.id} for ${leave.employee_name} has been rejected.`
        );
        delete this.comments[leave.id];
        this.fetchPendingApprovals();
      },
      error: (err) => {
        this.processingId.set(null);
        this.notify.error('Rejection Failed', err.error?.message || 'Failed to reject leave request.');
      }
    });
  }

  onFilterChange(): void {
    this.currentPage = 1;
    this.fetchPendingApprovals();
  }

  resetFilters(): void {
    this.typeFilter = '';
    this.fromDate = '';
    this.toDate = '';
    this.currentPage = 1;
    this.fetchPendingApprovals();
  }

  goToPage(page: number): void {
    if (page < 1 || page > this.pagination().total_pages) return;
    this.currentPage = page;
    this.fetchPendingApprovals();
  }

  viewDetails(leave: LeaveDetails): void {
    this.selectedLeave.set(leave);
  }

  closeModal(): void {
    this.selectedLeave.set(null);
  }
}
