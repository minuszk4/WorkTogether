import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { adminGuard } from './core/guards/admin.guard';

export const routes: Routes = [
  {
	path: 'admin',
	loadComponent: () => import('./features/admin/admin.component').then(m => m.AdminComponent),
	canActivate: [authGuard, adminGuard]
  },
  {
    path: '',
    redirectTo: 'dashboard',
    pathMatch: 'full'
  },
  {
    path: 'auth',
    loadComponent: () => import('./features/auth/auth.component').then(m => m.AuthComponent)
  },
  {
    path: 'auth/callback',
    loadComponent: () => import('./features/auth/auth-callback.component').then(m => m.AuthCallbackComponent)
  },
  {
    path: 'dashboard',
    loadComponent: () => import('./features/dashboard/dashboard.component').then(m => m.DashboardComponent),
    canActivate: [authGuard]
  },
  {
    path: 'room/:room_id',
    loadComponent: () => import('./features/room/room.component').then(m => m.RoomComponent),
    canActivate: [authGuard]
  },
  {
    path: '**',
    redirectTo: 'dashboard'
  }
];
