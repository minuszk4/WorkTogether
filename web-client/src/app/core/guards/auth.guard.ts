import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { StateService } from '../services/state.service';

export const authGuard: CanActivateFn = (route, state) => {
  const stateService = inject(StateService);
  const router = inject(Router);

  if (stateService.accessToken) {
    return true;
  }

  // Redirect to login page
  router.navigate(['/auth']);
  return false;
};
