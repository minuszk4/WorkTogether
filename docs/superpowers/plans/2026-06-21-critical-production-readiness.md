# Critical Production Readiness Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Secure the SyncSpace system and make it deployable by fixing CORS, query token logs, hardcoded localhosts, insecure fallbacks, local token vulnerability, and repository clutter.

**Architecture:** Use Nginx configuration logic for security mapping, Angular environment file swapping for frontend configurations, environment variable reading for backend redirects, startup assertion blocks in Go services, and Angular route guards for in-memory token refresh.

**Tech Stack:** Nginx, Go (Gin), Angular 19, Git.

## Global Constraints
- Target Node/Angular version: Node.js 18+, Angular 19+
- Target Go version: 1.21+
- No default insecure fallback values for secrets in production.
- Do not store access tokens in `localStorage`.

---

### Task 1: Nginx Gateway Configuration (CORS & Log Masking)

**Files:**
- Modify: `docker/nginx/nginx.conf`

- [ ] **Step 1: Update Nginx CORS and logging map configuration**
  We will add CORS whitelist verification and mask WS tokens from access logs.
  Edit `docker/nginx/nginx.conf` by adding the maps before the server block and updating headers and log format.
  
  ```nginx
  # Add under http block, above limit_req_zone:
  # CORS origin mapping
  map $http_origin $cors_origin {
      default "";
      "~^https?://localhost(:[0-9]+)?$" "$http_origin";
      "~^https?://127\.0\.0\.1(:[0-9]+)?$" "$http_origin";
  }

  # Mask token query parameters in logs
  map $request $masked_request {
      default $request;
      "~^(?<prefix>[^\?]+)\?token=[^&\s]+(?<suffix>.*)$" "${prefix}?token=***${suffix}";
      "~^(?<prefix>[^\?]+)\?(?<mid>.*)&token=[^&\s]+(?<suffix>.*)$" "${prefix}?${mid}&token=***${suffix}";
  }

  # Update log_format main
  log_format main '$remote_addr - $remote_user [$time_local] "$masked_request" '
                  '$status $body_bytes_sent "$http_referer" '
                  '"$http_x_forwarded_for"';
  ```

- [ ] **Step 2: Update CORS Headers in Nginx server block**
  Inside the `server` block of `docker/nginx/nginx.conf`, edit lines 40-43:
  ```nginx
  # Replace:
  # add_header 'Access-Control-Allow-Origin' '$http_origin' always;
  # With:
  add_header 'Access-Control-Allow-Origin' '$cors_origin' always;
  ```

- [ ] **Step 3: Verify Nginx syntax**
  Verify that Nginx configuration has no syntax issues (can be tested locally if Nginx is installed, or by running docker-compose config check).

---

### Task 2: Frontend Environment & API/WS Url upgrades (C2)

**Files:**
- Create: `web-client/src/environments/environment.ts`
- Create: `web-client/src/environments/environment.prod.ts`
- Modify: `web-client/angular.json`
- Modify: `web-client/src/app/core/services/api.service.ts`
- Modify: `web-client/src/app/core/interceptors/jwt.interceptor.ts`
- Modify: `web-client/src/app/core/services/websocket/chat-ws.service.ts`
- Modify: `web-client/src/app/core/services/websocket/playback-ws.service.ts`
- Modify: `web-client/src/app/features/auth/auth.component.ts`

- [ ] **Step 1: Create `environment.ts`**
  Write to `web-client/src/environments/environment.ts`:
  ```typescript
  export const environment = {
    production: false,
    apiUrl: 'http://localhost:8080/api/v1',
    wsUrl: 'ws://localhost:8080',
    livekitUrl: 'ws://localhost:7880'
  };
  ```

- [ ] **Step 2: Create `environment.prod.ts`**
  Write to `web-client/src/environments/environment.prod.ts`:
  ```typescript
  export const environment = {
    production: true,
    apiUrl: '/api/v1',
    wsUrl: (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host,
    livekitUrl: (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/voice'
  };
  ```

- [ ] **Step 3: Update `angular.json` configuration**
  Add the `fileReplacements` block in `web-client/angular.json` under `projects.web-client.architect.build.configurations.production`:
  ```json
  "fileReplacements": [
    {
      "replace": "src/environments/environment.ts",
      "with": "src/environments/environment.prod.ts"
    }
  ]
  ```

- [ ] **Step 4: Update API & Web Socket Service URLs**
  - In `api.service.ts`, import `environment` and replace the private `apiBase` field:
    ```typescript
    import { environment } from '../../../environments/environment';
    ...
    private apiBase = environment.apiUrl;
    ```
  - In `jwt.interceptor.ts`, import `environment` and replace line 56:
    ```typescript
    import { environment } from '../../../environments/environment';
    ...
    // Replace: 'http://localhost:8080/api/v1/auth/refresh'
    // With:
    environment.apiUrl + '/auth/refresh'
    ```
  - In `chat-ws.service.ts` and `playback-ws.service.ts`, import `environment` and replace the `wsUrl` string:
    ```typescript
    import { environment } from '../../../../environments/environment';
    ...
    const wsUrl = `${environment.wsUrl}/api/v1/rooms/${roomId}/chat/ws?token=${token}`; // (and playback/ws respectively)
    ```
  - In `auth.component.ts`, import `environment` and replace line 105:
    ```typescript
    import { environment } from '../../../environments/environment';
    ...
    window.location.href = `${environment.apiUrl}/auth/google`;
    ```

- [ ] **Step 5: Build frontend to verify compilation**
  Run command in `web-client` directory: `npm run build`

---

### Task 3: Backend Redirects (C2)

**Files:**
- Modify: `services/auth-service/cmd/server/main.go`
- Modify: `services/auth-service/internal/delivery/http/handlers.go`
- Modify: `services/auth-service/internal/usecase/email.go`
- Modify: `.env.example`
- Modify: `.env`

- [ ] **Step 1: Load `FRONTEND_URL` in `main.go`**
  Get `FRONTEND_URL` from the environment and pass it to UseCase or Handlers.
  Wait, the handers/email usecase is created in `main.go`:
  Let's check `services/auth-service/internal/delivery/http/handlers.go` constructor:
  ```go
  type AuthHandler struct {
      usecase    *usecase.AuthUsecase
      googleCfg  *oauth2.Config
      frontendURL string // We can add frontendURL field here!
  }
  ```
  And `NewAuthHandler`:
  ```go
  func NewAuthHandler(uc *usecase.AuthUsecase, gc *oauth2.Config, frontendURL string) *AuthHandler {
      return &AuthHandler{
          usecase:     uc,
          googleCfg:   gc,
          frontendURL: frontendURL,
      }
  }
  ```
  We also need to pass `FRONTEND_URL` to `emailSvc` or pass it directly in email template context.
  Let's look at `email.go` to see where `http://localhost:4200` is used.
  In `email.go`, it sends the activation email. We should modify `emailSvc.SendVerificationEmail` to accept a base frontend URL.

- [ ] **Step 2: Update redirects in `handlers.go`**
  Replace all occurrences of `http://localhost:4200` in `handlers.go` with `h.frontendURL`.
  Example for verify email redirect:
  ```go
  c.Redirect(http.StatusFound, h.frontendURL+"/auth?verified=true")
  ```

- [ ] **Step 3: Update `.env` and `.env.example`**
  Add `FRONTEND_URL=http://localhost:4200` to both files.

---

### Task 4: JWT secret fail-fast validation in all 11 Go services (C3)

**Files:**
- Modify `main.go` of:
  - `services/auth-service/cmd/server/main.go`
  - `services/chat-service/cmd/server/main.go`
  - `services/collab-service/cmd/server/main.go`
  - `services/music-service/cmd/server/main.go`
  - `services/notification-service/cmd/server/main.go`
  - `services/playback-service/cmd/server/main.go`
  - `services/playlist-service/cmd/server/main.go`
  - `services/room-service/cmd/server/main.go`
  - `services/timer-service/cmd/server/main.go`
  - `services/user-service/cmd/server/main.go`
  - `services/voice-service/cmd/server/main.go`

- [ ] **Step 1: Implement JWT_SECRET assertion in `main.go`**
  Modify all 11 services. Instead of:
  ```go
  jwtSecret := getEnv("JWT_SECRET", "worktogether_secret_key_12345")
  ```
  Use:
  ```go
  jwtSecret := os.Getenv("JWT_SECRET")
  if jwtSecret == "" {
      log.Fatal("FATAL: Environment variable JWT_SECRET is not set. Service cannot start.")
  }
  ```

---

### Task 5: Voice Service LiveKit secret fail-fast validation (M3)

**Files:**
- Modify: `services/voice-service/cmd/server/main.go`

- [ ] **Step 1: Check LiveKit credentials**
  In `services/voice-service/cmd/server/main.go`, replace:
  ```go
  livekitKey := getEnv("LIVEKIT_API_KEY", "devkey")
  livekitSecret := getEnv("LIVEKIT_API_SECRET", "worktogether_livekit_dev_secret_1234567890")
  ```
  With:
  ```go
  livekitKey := os.Getenv("LIVEKIT_API_KEY")
  livekitSecret := os.Getenv("LIVEKIT_API_SECRET")
  if livekitKey == "" || livekitSecret == "" {
      log.Fatal("FATAL: LiveKit API credentials (LIVEKIT_API_KEY / LIVEKIT_API_SECRET) are not configured.")
  }
  if livekitKey == "devkey" {
      log.Println("WARNING: Running voice-service with default LiveKit development credentials ('devkey')")
  }
  ```

---

### Task 6: In-memory token storage (C4) in frontend, including asynchronous routing refresh guard

**Files:**
- Modify: `web-client/src/app/core/services/state.service.ts`
- Modify: `web-client/src/app/core/guards/auth.guard.ts`

- [ ] **Step 1: Remove localStorage token persistence in `state.service.ts`**
  - Delete restoring `savedToken` from localStorage in constructor.
  - Delete `this.accessToken$.subscribe(...)` block that stores access token in localStorage.
  Keep `user$` storage in localStorage.

- [ ] **Step 2: Update `auth.guard.ts`**
  Implement asynchronous validation to perform a token refresh on startup / page reload if `stateService.accessToken` is empty.
  ```typescript
  import { inject } from '@angular/core';
  import { CanActivateFn, Router } from '@angular/router';
  import { StateService } from '../services/state.service';
  import { HttpClient } from '@angular/common/http';
  import { of, Observable } from 'rxjs';
  import { catchError, map } from 'rxjs/operators';
  import { environment } from '../services/api.service'; // Or import environment directly
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
  ```

---

### Task 7: Git cleanup & ignore build patterns (C6)

**Files:**
- Modify: `.gitignore`
- Command execution: Git commands

- [ ] **Step 1: Add build output ignore rules to `.gitignore`**
  Append to `.gitignore`:
  ```gitignore
  # Compiled Service Binaries
  *-linux
  *service-linux
  ```

- [ ] **Step 2: Run git rm command**
  Run: `git rm --cached services/chat-service/chat-service-linux`
