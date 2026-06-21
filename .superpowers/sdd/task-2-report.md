# Task 2: Frontend Environment & API/WS Url upgrades (C2) - Report

## What was implemented
1. **Created Development Environment Configuration (`web-client/src/environments/environment.ts`)**:
   - Configured `production: false`, API URL, WebSocket URL, and LiveKit URL to point to localhost.
2. **Created Production Environment Configuration (`web-client/src/environments/environment.prod.ts`)**:
   - Configured `production: true`, API URL, dynamic WebSocket URL based on `window.location.host`, and dynamic LiveKit URL.
3. **Updated Build Configuration (`web-client/angular.json`)**:
   - Added the `fileReplacements` array under the `production` build configuration to swap `environment.ts` with `environment.prod.ts`.
4. **Refactored Frontend URLs to use Dynamic Environment Variables**:
   - **`api.service.ts`**: Swapped hardcoded localhost base API URL with `environment.apiUrl`.
   - **`jwt.interceptor.ts`**: Swapped hardcoded token refresh URL with `environment.apiUrl + '/auth/refresh'`.
   - **`chat-ws.service.ts`**: Swapped hardcoded localhost chat websocket URL with `${environment.wsUrl}/api/v1/rooms/${roomId}/chat/ws...`.
   - **`playback-ws.service.ts`**: Swapped hardcoded localhost playback websocket URL with `${environment.wsUrl}/api/v1/rooms/${roomId}/playback/ws...`.
   - **`auth.component.ts`**: Swapped hardcoded localhost Google login URL with `${environment.apiUrl}/auth/google`.

## Verification check performed & Result
- Ran `npm run build` from the `web-client` directory.
- **Result**: The production bundle generation completed successfully without any compilation errors in 7.84 seconds.

## Files changed
- **New Files**:
  - `web-client/src/environments/environment.ts`
  - `web-client/src/environments/environment.prod.ts`
- **Modified Files**:
  - `web-client/angular.json`
  - `web-client/src/app/core/services/api.service.ts`
  - `web-client/src/app/core/interceptors/jwt.interceptor.ts`
  - `web-client/src/app/core/services/websocket/chat-ws.service.ts`
  - `web-client/src/app/core/services/websocket/playback-ws.service.ts`
  - `web-client/src/app/features/auth/auth.component.ts`

## Self-review findings
- The codebase follows standard Angular environments architecture.
- Replaced all localhost occurrences in the service layer as requested.
- Verification command output confirms the build compiles successfully.

## Issues or concerns
- None.
