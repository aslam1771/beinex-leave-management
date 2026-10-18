import { Injectable, computed, signal } from '@angular/core';
import { Employee } from '../models/employee.model';
import { ApiService } from './api.service';

// Seeded fallback employees matching db/seed.sql
export const SEEDED_EMPLOYEES: Employee[] = [
  {
    id: 1,
    employee_code: 'EMP-001',
    first_name: 'Alex',
    last_name: 'Taylor',
    email: 'alex.taylor@beinex.com',
    role: 'ADMIN',
    department: 'Human Resources',
    designation: 'HR Administrator',
    manager_id: null,
    joining_date: '2021-03-01',
    is_active: true
  },
  {
    id: 2,
    employee_code: 'EMP-002',
    first_name: 'Sarah',
    last_name: 'Connor',
    email: 'sarah.connor@beinex.com',
    role: 'MANAGER',
    department: 'Engineering',
    designation: 'Engineering Manager',
    manager_id: null,
    joining_date: '2021-06-15',
    is_active: true
  },
  {
    id: 3,
    employee_code: 'EMP-003',
    first_name: 'John',
    last_name: 'Doe',
    email: 'john.doe@beinex.com',
    role: 'EMPLOYEE',
    department: 'Engineering',
    designation: 'Senior Backend Developer',
    manager_id: 2,
    joining_date: '2022-01-10',
    is_active: true
  },
  {
    id: 4,
    employee_code: 'EMP-004',
    first_name: 'Jane',
    last_name: 'Smith',
    email: 'jane.smith@beinex.com',
    role: 'EMPLOYEE',
    department: 'Engineering',
    designation: 'Frontend Developer',
    manager_id: 2,
    joining_date: '2022-04-01',
    is_active: true
  },
  {
    id: 5,
    employee_code: 'EMP-005',
    first_name: 'Michael',
    last_name: 'Chang',
    email: 'michael.chang@beinex.com',
    role: 'EMPLOYEE',
    department: 'Engineering',
    designation: 'QA Automation Engineer',
    manager_id: 2,
    joining_date: '2022-09-20',
    is_active: true
  }
];

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  readonly employees = signal<Employee[]>(SEEDED_EMPLOYEES);
  readonly currentUser = signal<Employee>(SEEDED_EMPLOYEES[2]); // default to John Doe

  readonly isManager = computed(() => {
    const user = this.currentUser();
    return user.role === 'MANAGER' || user.role === 'ADMIN';
  });

  readonly isAdmin = computed(() => {
    return this.currentUser().role === 'ADMIN';
  });

  constructor(private api: ApiService) {
    this.refreshEmployees();
  }

  refreshEmployees(): void {
    this.api.getEmployees().subscribe({
      next: (emps) => {
        if (emps && emps.length > 0) {
          this.employees.set(emps);
          // Keep current selection if valid, else pick first employee
          const current = this.currentUser();
          const found = emps.find(e => e.id === current.id);
          if (found) {
            this.currentUser.set(found);
          } else {
            this.currentUser.set(emps[0]);
          }
        }
      },
      error: () => {
        // Fallback already preloaded
      }
    });
  }

  switchUser(employeeId: number): void {
    const selected = this.employees().find(e => e.id === employeeId);
    if (selected) {
      this.currentUser.set(selected);
    }
  }
}
