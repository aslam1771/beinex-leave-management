import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { AuthService } from './services/auth.service';
import { ApiService } from './services/api.service';
import { NotificationService } from './services/notification.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive, FormsModule],
  templateUrl: './app.html',
  styleUrls: ['./app.css']
})
export class App implements OnInit {
  readonly backendOnline = signal<boolean | null>(null);
  readonly managerPendingBadge = signal(0);
  readonly sidebarOpen = signal(true);

  constructor(
    public auth: AuthService,
    private api: ApiService,
    public notify: NotificationService
  ) {}

  ngOnInit(): void {
    this.checkHealth();
  }

  checkHealth(): void {
    this.api.getHealth().subscribe({
      next: (res) => {
        this.backendOnline.set(res && (res.status === 'healthy' || res.status === 'degraded'));
      },
      error: () => {
        this.backendOnline.set(false);
      }
    });
  }

  onUserSelect(event: Event): void {
    const target = event.target as HTMLSelectElement;
    if (target && target.value) {
      this.auth.switchUser(Number(target.value));
    }
  }

  toggleSidebar(): void {
    this.sidebarOpen.update(v => !v);
  }
}
