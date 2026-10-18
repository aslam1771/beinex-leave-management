import { Component, OnInit, effect, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { LeaveDetails, LeaveStatus, LeaveType, PaginationMeta } from '../../models/leave.model';

@Component({
  selector: 'app-my-leaves',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './my-leaves.component.html',
  styleUrls: ['./my-leaves.component.css']
})
export class MyLeavesComponent implements OnInit {
  readonly loading = signal(true);
  readonly cancellingId = signal<number | null>(null);
  readonly leaves = signal<LeaveDetails[]>([]);
  readonly pagination = signal<PaginationMeta>({ page: 1, limit: 10, total: 0, total_pages: 1 });

  // Filter state
  statusFilter = '';
  typeFilter = '';
  fromDate = '';
  toDate = '';
  currentPage = 1;
  pageSize = 10;

  // Selected leave for detail modal
  selectedLeave = signal<LeaveDetails | null>(null);

  constructor(
    public auth: AuthService,
    private api: ApiService,
    private notify: NotificationService
  ) {
    // Reload when user switches
    effect(() => {
      const user = this.auth.currentUser();
      if (user) {
        this.currentPage = 1;
        this.fetchLeaves();
      }
    });
  }

  ngOnInit(): void {
    this.fetchLeaves();
  }

  fetchLeaves(): void {
    const user = this.auth.currentUser();
    if (!user) return;

    this.loading.set(true);

    const filter = {
      status: (this.statusFilter || undefined) as LeaveStatus,
      leave_type: (this.typeFilter || undefined) as LeaveType,
      from_date: this.fromDate || undefined,
      to_date: this.toDate || undefined,
      page: this.currentPage,
      limit: this.pageSize
    };

    this.api.getEmployeeLeaves(user.id, filter).subscribe({
      next: (res) => {
        this.leaves.set(res.data || []);
        if (res.pagination) {
          this.pagination.set(res.pagination);
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.notify.error('Error', 'Failed to load leave history: ' + (err.error?.message || err.message));
      }
    });
  }

  onFilterChange(): void {
    this.currentPage = 1;
    this.fetchLeaves();
  }

  resetFilters(): void {
    this.statusFilter = '';
    this.typeFilter = '';
    this.fromDate = '';
    this.toDate = '';
    this.currentPage = 1;
    this.fetchLeaves();
  }

  goToPage(page: number): void {
    if (page < 1 || page > this.pagination().total_pages) return;
    this.currentPage = page;
    this.fetchLeaves();
  }

  cancelRequest(leave: LeaveDetails): void {
    if (!confirm(`Are you sure you want to cancel your ${leave.leave_type} leave request (#${leave.id})?`)) {
      return;
    }

    const user = this.auth.currentUser();
    if (!user) return;

    this.cancellingId.set(leave.id);

    this.api.cancelLeave(leave.id, { employee_id: user.id }).subscribe({
      next: () => {
        this.cancellingId.set(null);
        this.notify.success('Cancelled', `Leave request #${leave.id} has been cancelled.`);
        this.fetchLeaves();
      },
      error: (err) => {
        this.cancellingId.set(null);
        this.notify.error('Cancellation Failed', err.error?.message || 'Could not cancel leave request.');
      }
    });
  }

  viewDetails(leave: LeaveDetails): void {
    this.selectedLeave.set(leave);
  }

  closeModal(): void {
    this.selectedLeave.set(null);
  }
}
