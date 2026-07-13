import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { StateService } from '../services/state.service';

export const adminGuard: CanActivateFn = () => {
  const token = inject(StateService).accessToken;
  const router = inject(Router);
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
    return payload.is_admin === true || router.createUrlTree(['/dashboard']);
  } catch {
    return router.createUrlTree(['/dashboard']);
  }
};
