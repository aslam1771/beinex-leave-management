export type EmployeeRole = 'ADMIN' | 'MANAGER' | 'EMPLOYEE';

export interface Employee {
  id: number;
  employee_code: string;
  first_name: string;
  last_name: string;
  email: string;
  role: EmployeeRole;
  department: string;
  designation: string;
  manager_id?: number | null;
  joining_date: string;
  is_active: boolean;
}
