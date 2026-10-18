import { Component, OnInit, effect, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { LeaveDetails } from '../../models/leave.model';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {
  readonly loading = signal(true);
  readonly leaves = signal<LeaveDetails[]>([]);
  readonly pendingCount = signal(0);
  readonly approvedCount = signal(0);
  readonly rejectedCount = signal(0);
  readonly totalDaysApproved = signal(0);
  readonly managerPendingCount = signal(0);

  constructor(
    public auth: AuthService,
    private api: ApiService
  ) {
    // Automatically re-fetch dashboard when active demo user switches
    effect(() => {
      const user = this.auth.currentUser();
      if (user) {
        this.loadDashboardData();
      }
    });
  }

  ngOnInit(): void {
    this.loadDashboardData();
  }

  loadDashboardData(): void {
    const user = this.auth.currentUser();
    if (!user) return;

    this.loading.set(true);

    // 1. Fetch current employee's leaves
    this.api.getEmployeeLeaves(user.id, { limit: 50 }).subscribe({
      next: (resp) => {
        const list = resp.data || [];
        this.leaves.set(list);

        let pending = 0;
        let approved = 0;
        let rejected = 0;
        let days = 0;

        for (const item of list) {
          if (item.status === 'PENDING') pending++;
          else if (item.status === 'APPROVED') {
            approved++;
            days += item.total_days;
          } else if (item.status === 'REJECTED' || item.status === 'CANCELLED') {
            rejected++;
          }
        }

        this.pendingCount.set(pending);
        this.approvedCount.set(approved);
        this.rejectedCount.set(rejected);
        this.totalDaysApproved.set(days);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
      }
    });

    // 2. If Manager or Admin, fetch pending approvals count
    if (this.auth.isManager()) {
      this.api.getManagerPendingLeaves(user.id, { limit: 1 }).subscribe({
        next: (resp) => {
          this.managerPendingCount.set(resp.pagination?.total || 0);
        },
        error: () => {
          this.managerPendingCount.set(0);
        }
      });
    } else {
      this.managerPendingCount.set(0);
    }
  }

  recentLeaves(): LeaveDetails[] {
    return this.leaves().slice(0, 5);
  }
}
