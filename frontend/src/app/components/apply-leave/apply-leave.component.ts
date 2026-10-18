import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { LeaveType } from '../../models/leave.model';

@Component({
  selector: 'app-apply-leave',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink],
  templateUrl: './apply-leave.component.html',
  styleUrls: ['./apply-leave.component.css']
})
export class ApplyLeaveComponent implements OnInit {
  leaveForm!: FormGroup;
  readonly submitting = signal(false);
  readonly errorMessage = signal<string | null>(null);
  readonly calculatedDays = signal<number | null>(null);

  readonly leaveTypes: { value: LeaveType; label: string; desc: string }[] = [
    { value: 'ANNUAL', label: 'Annual Leave', desc: 'Paid vacation time' },
    { value: 'SICK', label: 'Sick Leave', desc: 'Medical and health recovery' },
    { value: 'CASUAL', label: 'Casual Leave', desc: 'Personal errands and urgent needs' },
    { value: 'MATERNITY', label: 'Maternity Leave', desc: 'New mother parental leave' },
    { value: 'PATERNITY', label: 'Paternity Leave', desc: 'New father parental leave' },
    { value: 'UNPAID', label: 'Unpaid Leave', desc: 'Leave without salary' }
  ];

  constructor(
    private fb: FormBuilder,
    private api: ApiService,
    public auth: AuthService,
    private notify: NotificationService,
    private router: Router
  ) {}

  ngOnInit(): void {
    const today = new Date().toISOString().split('T')[0];

    this.leaveForm = this.fb.group({
      leave_type: ['ANNUAL', Validators.required],
      start_date: [today, Validators.required],
      end_date: [today, Validators.required],
      total_days: [1, [Validators.required, Validators.min(0.5)]],
      reason: ['', [Validators.required, Validators.minLength(5)]]
    });

    this.updateDuration();

    // Recalculate duration on date changes
    this.leaveForm.get('start_date')?.valueChanges.subscribe(() => this.updateDuration());
    this.leaveForm.get('end_date')?.valueChanges.subscribe(() => this.updateDuration());
  }

  updateDuration(): void {
    const startStr = this.leaveForm.get('start_date')?.value;
    const endStr = this.leaveForm.get('end_date')?.value;

    if (!startStr || !endStr) {
      this.calculatedDays.set(null);
      return;
    }

    const start = new Date(startStr);
    const end = new Date(endStr);

    if (end < start) {
      this.calculatedDays.set(0);
      return;
    }

    const diffTime = end.getTime() - start.getTime();
    const diffDays = Math.round(diffTime / (1000 * 60 * 60 * 24)) + 1;

    this.calculatedDays.set(diffDays);
    this.leaveForm.patchValue({ total_days: diffDays }, { emitEvent: false });
  }

  onSubmit(): void {
    this.errorMessage.set(null);

    if (this.leaveForm.invalid) {
      this.leaveForm.markAllAsTouched();
      return;
    }

    const startStr = this.leaveForm.value.start_date;
    const endStr = this.leaveForm.value.end_date;
    if (new Date(endStr) < new Date(startStr)) {
      this.errorMessage.set('End date cannot be earlier than start date.');
      return;
    }

    const currentUser = this.auth.currentUser();
    if (!currentUser) {
      this.errorMessage.set('No active employee selected in demo profile.');
      return;
    }

    this.submitting.set(true);

    const payload = {
      employee_id: currentUser.id,
      leave_type: this.leaveForm.value.leave_type,
      start_date: startStr,
      end_date: endStr,
      total_days: Number(this.leaveForm.value.total_days),
      reason: this.leaveForm.value.reason.trim()
    };

    this.api.applyLeave(payload).subscribe({
      next: (res) => {
        this.submitting.set(false);
        this.notify.success(
          'Application Submitted!',
          `Your ${res.leave_type} leave request for ${res.total_days} day(s) was submitted successfully.`
        );
        this.router.navigate(['/my-leaves']);
      },
      error: (err) => {
        this.submitting.set(false);
        const serverError = err.error?.message || err.error?.error || 'Failed to submit leave request.';
        this.errorMessage.set(serverError);
      }
    });
  }
}
