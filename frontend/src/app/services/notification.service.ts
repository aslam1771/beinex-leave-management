import { Injectable, signal } from '@angular/core';

export interface AppNotification {
  id: number;
  type: 'success' | 'error' | 'info';
  title: string;
  message: string;
}

@Injectable({
  providedIn: 'root'
})
export class NotificationService {
  private nextId = 1;
  readonly notifications = signal<AppNotification[]>([]);

  show(type: 'success' | 'error' | 'info', title: string, message: string, durationMs = 5000): void {
    const id = this.nextId++;
    const item: AppNotification = { id, type, title, message };
    this.notifications.update(list => [...list, item]);

    if (durationMs > 0) {
      setTimeout(() => {
        this.remove(id);
      }, durationMs);
    }
  }

  success(title: string, message: string): void {
    this.show('success', title, message);
  }

  error(title: string, message: string): void {
    this.show('error', title, message, 7000);
  }

  info(title: string, message: string): void {
    this.show('info', title, message);
  }

  remove(id: number): void {
    this.notifications.update(list => list.filter(n => n.id !== id));
  }
}
