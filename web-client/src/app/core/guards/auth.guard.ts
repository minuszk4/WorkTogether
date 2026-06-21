import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { StateService } from '../services/state.service';
import { HttpClient } from '@angular/common/http';
import { of, Observable } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { environment as env } from '../../../environments/environment';

export const authGuard: CanActivateFn = (route, state): Observable<boolean> | boolean => {
  const stateService = inject(StateService);
  const router = inject(Router);
  const http = inject(HttpClient);

  if (stateService.accessToken) {
    return true;
  }

  return http.post<any>(`${env.apiUrl}/auth/refresh`, {}, { withCredentials: true }).pipe(
    map((res: any) => {
      if (res?.success && res?.data?.access_token) {
        stateService.accessToken$.next(res.data.access_token);
        return true;
      }
      router.navigate(['/auth']);
      return false;
    }),
    catchError(() => {
      router.navigate(['/auth']);
      return of(false);
    })
  );
};

