# Task 1 Report: Nginx Gateway Configuration (CORS & Log Masking)

## What Was Implemented
We have updated the Nginx Gateway Configuration (`docker/nginx/nginx.conf`) to implement CORS protection and log masking for JWT tokens.

1. **CORS Origin Validation**:
   - Added a CORS origin mapping using `map $http_origin $cors_origin` to whitelist only `localhost` and `127.0.0.1` (with optional ports).
   - Replaced `Access-Control-Allow-Origin: $http_origin` with `Access-Control-Allow-Origin: $cors_origin` inside the `server` block.

2. **Log Masking**:
   - Added regex mapping (`map $request $masked_request`) to intercept request URLs containing a `token=` query parameter and replace the token value with `***`.
   - Updated the `log_format main` to log `$masked_request` instead of `$request`.
   - Fixed a bug in the default log format where `$http_x_forwarded_for` was missing the `$` prefix.

## Verification Check Performed
- Ran `docker compose config` to verify the Docker Compose file validates successfully.
- *Note on Nginx syntax test*: The Docker daemon is not running on the local host (failed to connect to local named pipe `//./pipe/dockerDesktopLinuxEngine`), so a live container-based syntax test (`nginx -t`) was not executable. However, the configurations were manually verified to match the Nginx declarative syntax standard for `map`, `log_format`, and `add_header` instructions.

## Files Changed
- `docker/nginx/nginx.conf`

## Self-Review Findings
- The changes strictly align with the task specification.
- Handled both cases for token position (at the start of query parameters and mid-query parameters).
- Clean separation of the map configurations, placed cleanly in the `http` block above the limit request zones.
