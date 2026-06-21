# Task 2 Report: Room Mute/Unmute & stand-alone Deletion (`room-service`)

## What Was Implemented

1. **Usecase Layer (`services/room-service/internal/usecase/room.go`):**
   - Implemented `DeleteRoom(ctx, userID, roomID)`: Validates that the requester is the `OWNER` of the room, and if so, deletes the room using repository `DeleteRoom`.
   - Implemented `MuteMember(ctx, requesterID, roomID, targetUserID, durationSecs)`: Validates that the requester is either the `OWNER` or a `MODERATOR`. Ensures the target member is not the `OWNER`. Calculates the duration and updates `muted_until` in the database via repository `UpdateMemberMute`.
   - Implemented `UnmuteMember(ctx, requesterID, roomID, targetUserID)`: Validates requester privileges (`OWNER` or `MODERATOR`) and clears the target member's `muted_until` status by passing a `nil` timestamp to repository `UpdateMemberMute`.

2. **REST delivery Layer (`services/room-service/internal/delivery/http/handlers.go`):**
   - Added REST handlers for:
     - `DeleteRoom` (`DELETE /api/v1/rooms/:id`)
     - `MuteMember` (`POST /api/v1/rooms/:id/members/:user_id/mute`)
     - `UnmuteMember` (`POST /api/v1/rooms/:id/members/:user_id/unmute`)

3. **Routing Configuration (`services/room-service/cmd/server/main.go`):**
   - Registered the `DELETE` route for room deletion under the authorized `roomsGroup`.
   - Registered `POST` routes for muting and unmuting members.

4. **gRPC delivery Layer (`services/room-service/internal/delivery/grpc/server.go`):**
   - Updated `VerifyRoomMember` to filter out permissions `CAN_CHAT` and `CAN_USE_VOICE` if the user's `MutedUntil` time is in the future.

5. **Unit Tests (`services/room-service/internal/usecase/room_test.go`):**
   - Added `sqlmock` tests testing success and unauthorized paths for `DeleteRoom`, `MuteMember`, and `UnmuteMember`.

---

## Verification Performed & Results

- **Unit Tests:** Ran `go test -v ./internal/usecase/...` inside `services/room-service`.
  - Result: All tests passed successfully.
    ```
    === RUN   TestGenerateInviteCode
    --- PASS: TestGenerateInviteCode (0.00s)
    === RUN   TestDeleteRoom
    === RUN   TestDeleteRoom/Success_as_Owner
    === RUN   TestDeleteRoom/Unauthorized_as_Member
    --- PASS: TestDeleteRoom (0.00s)
        --- PASS: TestDeleteRoom/Success_as_Owner (0.00s)
        --- PASS: TestDeleteRoom/Unauthorized_as_Member (0.00s)
    === RUN   TestMuteMember
    === RUN   TestMuteMember/Success_Mute
    === RUN   TestMuteMember/Cannot_Mute_Owner
    --- PASS: TestMuteMember (0.00s)
        --- PASS: TestMuteMember/Success_Mute (0.00s)
        --- PASS: TestMuteMember/Cannot_Mute_Owner (0.00s)
    === RUN   TestUnmuteMember
    === RUN   TestUnmuteMember/Success_Unmute
    --- PASS: TestUnmuteMember (0.00s)
        --- PASS: TestUnmuteMember/Success_Unmute (0.00s)
    PASS
    ok  	github.com/worktogether/services/room-service/internal/usecase	1.222s
    ```
- **Service compilation:** Successfully built the room-service using `go build ./cmd/server`.

---

## Files Changed

- `services/room-service/internal/usecase/room.go`
- `services/room-service/internal/usecase/room_test.go`
- `services/room-service/internal/delivery/http/handlers.go`
- `services/room-service/internal/delivery/grpc/server.go`
- `services/room-service/cmd/server/main.go`
- `services/room-service/go.mod`
- `services/room-service/go.sum`

---

## Self-Review Findings

- Used robust `c.GetString("userID")` extraction to fetch the userID from Gin context securely and type-safely.
- Leveraged existing `UpdateMemberMute` method in Postgres repository, preserving database schema integrity.
- Correctly parsed the `MuteRequest` structure's binding and handled duration validation.
- Kept the custom roles and permissions override logic clean and fully compatible with gRPC's VerifyRoomMember flow.

---

## Issues or Concerns

- None.

---

## Fixes Implemented (Review Feedback)

The following changes were made to resolve review feedback issues:

1. **Non-compliant JSON response structure in HTTP handlers:**
   - Updated `DeleteRoom`, `MuteMember`, and `UnmuteMember` handlers in [handlers.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/room-service/internal/delivery/http/handlers.go) to return the unified JSON response format.

2. **Improper HTTP Status Codes on Usecase Errors:**
   - Mapped `errors.Is(err, usecase.ErrUnauthorized)` to `http.StatusForbidden` (403) with message `"Không có quyền thực hiện."`.
   - Mapped target member non-existence (`"thành viên mục tiêu không tồn tại"`) to `http.StatusNotFound` (404).
   - Mapped attempting to mute a room owner (`"không thể mute chủ phòng"`) to `http.StatusBadRequest` (400).
   - Standardized parameter/JSON binding failures to `http.StatusBadRequest` (400) with code `INVALID_PARAMETERS`.
   - Standardized database or other errors to `http.StatusInternalServerError` (500) with code `ROOM_ERROR`.

3. **Missing mock.ExpectationsWereMet() in TestMuteMember:**
   - Added `if err := mock.ExpectationsWereMet(); err != nil { t.Errorf(...) }` check to the `Cannot Mute Owner` subtest in [room_test.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/room-service/internal/usecase/room_test.go).

4. **Decouple Error Checking in HTTP Handlers & Propagate Infrastructure Errors:**
   - Defined package-level sentinel errors `ErrTargetMemberNotFound` and `ErrCannotMuteOwner` in [room.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/room-service/internal/usecase/room.go).
   - Modified `MuteMember` and `UnmuteMember` methods to return these sentinel errors instead of raw errors created by `errors.New(...)`.
   - Modified HTTP handlers in [handlers.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/room-service/internal/delivery/http/handlers.go) to use `errors.Is(err, ...)` for checking target member existence and mute restrictions on room owners.
   - Refactored `GetMember` checks in `MuteMember` and `UnmuteMember` to directly propagate infrastructure/database errors (when `err != nil`) instead of masking them under `ErrUnauthorized`.
   - Added new test cases in [room_test.go](file:///c:/Users/tranv/Desktop/WorkTogether/services/room-service/internal/usecase/room_test.go) to cover target member not found scenarios and database connection failure propagation.

### Verification
- Ran usecase tests:
  ```bash
  go test -v ./internal/usecase/...
  ```
  Output: All 11 tests (including the new database error and member not found subtests) passed successfully:
  ```
  === RUN   TestGenerateInviteCode
  --- PASS: TestGenerateInviteCode (0.00s)
  === RUN   TestDeleteRoom
  === RUN   TestDeleteRoom/Success_as_Owner
  === RUN   TestDeleteRoom/Unauthorized_as_Member
  --- PASS: TestDeleteRoom (0.00s)
      --- PASS: TestDeleteRoom/Success_as_Owner (0.00s)
      --- PASS: TestDeleteRoom/Unauthorized_as_Member (0.00s)
  === RUN   TestMuteMember
  === RUN   TestMuteMember/Success_Mute
  === RUN   TestMuteMember/Cannot_Mute_Owner
  === RUN   TestMuteMember/Target_Member_Not_Found
  === RUN   TestMuteMember/Database_error_on_requester_check
  === RUN   TestMuteMember/Database_error_on_target_check
  --- PASS: TestMuteMember (0.00s)
      --- PASS: TestMuteMember/Success_Mute (0.00s)
      --- PASS: TestMuteMember/Cannot_Mute_Owner (0.00s)
      --- PASS: TestMuteMember/Target_Member_Not_Found (0.00s)
      --- PASS: TestMuteMember/Database_error_on_requester_check (0.00s)
      --- PASS: TestMuteMember/Database_error_on_target_check (0.00s)
  === RUN   TestUnmuteMember
  === RUN   TestUnmuteMember/Success_Unmute
  === RUN   TestUnmuteMember/Target_Member_Not_Found
  === RUN   TestUnmuteMember/Database_error_on_requester_check
  === RUN   TestUnmuteMember/Database_error_on_target_check
  --- PASS: TestUnmuteMember (0.00s)
      --- PASS: TestUnmuteMember/Success_Unmute (0.00s)
      --- PASS: TestUnmuteMember/Target_Member_Not_Found (0.00s)
      --- PASS: TestUnmuteMember/Database_error_on_requester_check (0.00s)
      --- PASS: TestUnmuteMember/Database_error_on_target_check (0.00s)
  PASS
  ok  	github.com/worktogether/services/room-service/internal/usecase	1.054s
  ```
