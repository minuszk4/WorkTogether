# Design Spec: Critical Production Readiness Fixes (Phase 1)

This document details the architectural design and modifications needed to resolve the 6 Critical vulnerabilities identified in the production readiness audit of SyncSpace.

---

## 1. CORS Configuration (C1)

### Current Problem
`docker/nginx/nginx.conf` reflects any origin (`$http_origin`) while setting `Access-Control-Allow-Credentials` to `true`. This allows any malicious website to execute authenticated requests against the API gateway (CSRF/Credential theft).

### Proposed Design
We will use Nginx's `map` block to validate the request's origin against a strict whitelist of allowed origins (e.g., `localhost` and `127.0.0.1` for development, and the eventual production domains). If the origin matches, it is allowed; otherwise, it is blocked.

#### Changes in `docker/nginx/nginx.conf`
```nginx
# Whitelist origins for CORS
map $http_origin $cors_origin {
    default "";
    "~^https?://localhost(:[0-9]+)?$" "$http_origin";
    "~^https?://127\.0\.0\.1(:[0-9]+)?$" "$http_origin";
    # Production domains whitelist can be appended here
}

server {
    ...
    # Replace reflected origin with whitelist variable
    add_header 'Access-Control-Allow-Origin' '$cors_origin' always;
    add_header 'Access-Control-Allow-Credentials' 'true' always;
}
```

---

## 2. Hardcoded Localhost & Environment Injection (C2)

### Current Problem
Multiple endpoints, web socket URLs, and redirect addresses are hardcoded to `localhost` in both frontend Angular services and the backend `auth-service` redirects.

### Proposed Design
We will introduce Angular environments, replace all hardcoded endpoints in the frontend, and add a dynamic environment variable `FRONTEND_URL` in the backend `auth-service` to drive redirects.

#### A. Frontend Environments & Angular CLI configuration
We will create environment files under `web-client/src/environments/`:

##### `web-client/src/environments/environment.ts` (Dev)
```typescript
export const environment = {
  production: false,
  apiUrl: 'http://localhost:8080/api/v1',
  wsUrl: 'ws://localhost:8080',
  livekitUrl: 'ws://localhost:7880'
};
```

##### `web-client/src/environments/environment.prod.ts` (Prod)
```typescript
export const environment = {
  production: true,
  apiUrl: '/api/v1', // Relative in production, terminated at Nginx Gateway
  wsUrl: 'wss://' + window.location.host, // Dynamic production WebSocket protocol
  livekitUrl: 'wss://' + window.location.host + '/voice'
};
```

##### `web-client/angular.json`
We will configure `fileReplacements` to load `environment.prod.ts` during production builds.
```json
"configurations": {
  "production": {
    "fileReplacements": [
      {
        "replace": "src/environments/environment.ts",
        "with": "src/environments/environment.prod.ts"
      }
    ],
    ...
  }
}
```

#### B. Replace Frontend Hardcoded References
- **`api.service.ts`**: Replace `private apiBase = 'http://localhost:8080/api/v1'` with `private apiBase = environment.apiUrl`.
- **`jwt.interceptor.ts`**: Replace `http://localhost:8080/api/v1/auth/refresh` with `environment.apiUrl + '/auth/refresh'`.
- **`chat-ws.service.ts`**: Replace `ws://localhost:8080/api/v1/...` with `environment.wsUrl + '/api/v1/...'`.
- **`playback-ws.service.ts`**: Replace `ws://localhost:8080/api/v1/...` with `environment.wsUrl + '/api/v1/...'`.
- **`auth.component.ts`**: Replace Google OAuth redirect `http://localhost:8080/api/v1/auth/google` with `${environment.apiUrl}/auth/google`.

#### C. Backend Redirects Configuration
In `services/auth-service/cmd/server/main.go`, read `FRONTEND_URL` from the environment.
Update `services/auth-service/internal/delivery/http/handlers.go` and `services/auth-service/internal/usecase/email.go` to use this config.

```go
// In handlers.go / email.go
frontendURL := getEnv("FRONTEND_URL", "http://localhost:4200")
c.Redirect(http.StatusFound, frontendURL + "/auth?verified=true")
```

---

## 3. JWT & Dependency Secret Fallback (C3 & M3)

### Current Problem
All microservices currently fallback to a public development JWT key if `JWT_SECRET` is missing. This could allow an attacker to forge valid tokens if the environment variable is misconfigured in production. Additionally, LiveKit dev credentials fallback is present in the `voice-service`.

### Proposed Design
We will remove the default fallback value for `JWT_SECRET` in all 11 services. If the variable is empty on startup, the application will terminate immediately (`log.Fatal`). We will do the same for the LiveKit credentials in the `voice-service`.

#### Main change pattern in `cmd/server/main.go` for all 11 services:
```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
}
```

#### Voice Service additions:
```go
livekitKey := os.Getenv("LIVEKIT_API_KEY")
livekitSecret := os.Getenv("LIVEKIT_API_SECRET")
if livekitKey == "" || livekitSecret == "" {
    log.Fatal("FATAL: LiveKit API credentials are not configured.")
}
```

---

## 4. In-Memory Access Token Storage (C4)

### Current Problem
The `accessToken` is stored in `localStorage` in the frontend `StateService`, which makes it vulnerable to theft via Cross-Site Scripting (XSS) attacks.

### Proposed Design
Access tokens should only live in-memory (using `BehaviorSubject`). We will remove the `localStorage` getters, setters, and initializers for `accessToken` in `StateService`.

To prevent logging out users on page refreshes, we will update `authGuard` to asynchronously request a token refresh using the HTTP-only `refresh_token` cookie before deciding to redirect to the login screen.

#### Changes in `auth.guard.ts`:
```typescript
export const authGuard: CanActivateFn = (route, state): Observable<boolean> | boolean => {
  const stateService = inject(StateService);
  const router = inject(Router);
  const http = inject(HttpClient);

  if (stateService.accessToken) {
    return true;
  }

  // Refresh token on page refresh
  return http.post<any>(`${environment.apiUrl}/auth/refresh`, {}, { withCredentials: true }).pipe(
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
```

---

## 5. WebSocket Token Logging Protection (C5)

### Current Problem
WebSockets pass the JWT token inside the URL query string (`?token=...`). Nginx logs the complete request line by default, exposing the token in access logs.

### Proposed Design
We will configure Nginx's log mapping regex to mask the `token` query parameter in the logged request variable `$masked_request`, ensuring it is stored in logs as `token=***`.

#### Changes in `docker/nginx/nginx.conf`:
```nginx
# Mask token query parameters in request logs
map $request $masked_request {
    default $request;
    "~^(?<prefix>[^\?]+)\?token=[^&\s]+(?<suffix>.*)$" "${prefix}?token=***${suffix}";
    "~^(?<prefix>[^\?]+)\?(?<mid>.*)&token=[^&\s]+(?<suffix>.*)$" "${prefix}?${mid}&token=***${suffix}";
}

# Update Nginx log_format
log_format main '$remote_addr - $remote_user [$time_local] "$masked_request" '
                '$status $body_bytes_sent "$http_referer" '
                '"$http_x_forwarded_for"';
```

---

## 6. Git Cleanup & Ignore patterns (C6)

### Current Problem
A compiled binary `services/chat-service/chat-service-linux` is currently tracked by git, cluttering the repository.

### Proposed Design
1. Run `git rm --cached services/chat-service/chat-service-linux` to remove the binary from the index.
2. Add binary build ignores (e.g., `*-linux`, `*service-linux`) to the root `.gitignore` to prevent future build outputs from being tracked.
