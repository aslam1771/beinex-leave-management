import { Routes } from '@angular/router';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { ApplyLeaveComponent } from './components/apply-leave/apply-leave.component';
import { MyLeavesComponent } from './components/my-leaves/my-leaves.component';
import { ManagerApprovalsComponent } from './components/manager-approvals/manager-approvals.component';

export const routes: Routes = [
  { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
  { path: 'dashboard', component: DashboardComponent },
  { path: 'apply', component: ApplyLeaveComponent },
  { path: 'my-leaves', component: MyLeavesComponent },
  { path: 'approvals', component: ManagerApprovalsComponent },
  { path: '**', redirectTo: 'dashboard' }
];
