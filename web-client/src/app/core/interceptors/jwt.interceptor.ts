import { HttpClient, HttpErrorResponse, HttpEvent, HttpHandlerFn, HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { catchError, finalize, shareReplay, switchMap } from 'rxjs/operators';
import { StateService } from '../services/state.service';

let refreshTokenRequest$: Observable<string> | null = null;

export const jwtInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {
  const stateService = inject(StateService);
  const router = inject(Router);
  const http = inject(HttpClient);

  const token = stateService.accessToken;
  let authReq = req;

  if (token) {
    authReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }

  authReq = authReq.clone({
    withCredentials: true
  });

  return next(authReq).pipe(
    catchError((error) => {
      if (
        error instanceof HttpErrorResponse &&
        error.status === 401 &&
        !authReq.url.includes('/auth/login') &&
        !authReq.url.includes('/auth/register') &&
        !authReq.url.includes('/auth/refresh')
      ) {
        return handle401Error(authReq, next, stateService, router, http);
      }

      return throwError(() => error);
    })
  );
};

function handle401Error(
  request: HttpRequest<unknown>,
  next: HttpHandlerFn,
  stateService: StateService,
  router: Router,
  http: HttpClient
): Observable<HttpEvent<unknown>> {
  if (!refreshTokenRequest$) {
    refreshTokenRequest$ = http
      .post<any>('http://localhost:8080/api/v1/auth/refresh', {}, { withCredentials: true })
      .pipe(
        switchMap((res: any) => {
          if (res?.success && res?.data?.access_token) {
            const newToken = res.data.access_token;
            stateService.accessToken$.next(newToken);
            return of(newToken);
          }

          stateService.clear();
          void router.navigate(['/auth']);
          return throwError(() => new Error('Refresh token failed'));
        }),
        catchError((err) => {
          stateService.clear();
          void router.navigate(['/auth']);
          return throwError(() => err);
        }),
        finalize(() => {
          refreshTokenRequest$ = null;
        }),
        shareReplay(1)
      );
  }

  return refreshTokenRequest$.pipe(
    switchMap((token) =>
      next(request.clone({
        setHeaders: {
          Authorization: `Bearer ${token}`
        }
      }))
    )
  );
}
