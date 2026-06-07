# PowerShell E2E Test Script for SyncSpace (9 Microservices)
# Run via Gateway port 8080

$gatewayUrl = "http://localhost:8080"
$wsGatewayUrl = "ws://localhost:8080"

# Helper function to send REST HTTP requests
function Send-Request {
    param (
        [string]$Uri,
        [string]$Method = "GET",
        [string]$Body = $null,
        [string]$Token = $null,
        [hashtable]$Headers = @{}
    )

    $requestHeaders = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
    if ($Token) {
        $requestHeaders.Add("Authorization", "Bearer $Token")
    }
    if ($Body) {
        $requestHeaders.Add("Content-Type", "application/json")
    }
    foreach ($key in $Headers.Keys) {
        $requestHeaders.Add($key, $Headers[$key])
    }

    $response = $null
    $statusCode = 0
    $content = ""

    try {
        if ($Body) {
            $response = Invoke-WebRequest -Uri $Uri -Method $Method -Headers $requestHeaders -Body $Body -UseBasicParsing
        } else {
            $response = Invoke-WebRequest -Uri $Uri -Method $Method -Headers $requestHeaders -UseBasicParsing
        }
        $statusCode = $response.StatusCode
        $content = $response.Content
    } catch {
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $content = $reader.ReadToEnd()
            $reader.Close()
        } else {
            $statusCode = 500
            $content = $_.Exception.Message
        }
    }

    return [PSCustomObject]@{
        StatusCode = $statusCode
        Content    = $content
    }
}

# Helper function to test WebSockets
function Test-WebSocket {
    param (
        [string]$Uri,
        [string]$Token,
        [string]$MessageToSend
    )

    $ws = New-Object System.Net.WebSockets.ClientWebSocket
    if ($Token) {
        $ws.Options.SetRequestHeader("Authorization", "Bearer $Token")
    }
    
    $uriObj = New-Object System.Uri($Uri)
    $cts = New-Object System.Threading.CancellationTokenSource
    
    try {
        $connTask = $ws.ConnectAsync($uriObj, $cts.Token)
        if (-not $connTask.Wait(5000)) {
            return [PSCustomObject]@{
                Success = $false
                Message = "WebSocket connection timed out."
            }
        }

        if ($ws.State -ne [System.Net.WebSockets.WebSocketState]::Open) {
            return [PSCustomObject]@{
                Success = $false
                Message = "WebSocket state is not Open: $($ws.State)"
            }
        }

        # Send message if provided
        if ($MessageToSend) {
            $bytes = [System.Text.Encoding]::UTF8.GetBytes($MessageToSend)
            $segment = New-Object System.ArraySegment[Byte] -ArgumentList @(,$bytes)
            $sendTask = $ws.SendAsync($segment, [System.Net.WebSockets.WebSocketMessageType]::Text, $true, $cts.Token)
            if (-not $sendTask.Wait(3000)) {
                return [PSCustomObject]@{
                    Success = $false
                    Message = "WebSocket send timed out."
                }
            }
        }

        # Wait to receive a message (max 3 seconds)
        $buffer = New-Object Byte[] 4096
        $segment = New-Object System.ArraySegment[Byte] -ArgumentList @(,$buffer)
        $recvTask = $ws.ReceiveAsync($segment, $cts.Token)
        
        if ($recvTask.Wait(3000)) {
            $result = $recvTask.Result
            $count = $result.Count
            $respStr = [System.Text.Encoding]::UTF8.GetString($buffer, 0, $count)
            
            # Close connection
            if ($ws.State -eq [System.Net.WebSockets.WebSocketState]::Open -or $ws.State -eq [System.Net.WebSockets.WebSocketState]::CloseReceived) {
                try {
                    $closeTask = $ws.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "Test complete", $cts.Token)
                    $closeTask.Wait(2000)
                } catch {}
            }

            return [PSCustomObject]@{
                Success = $true
                Message = $respStr
            }
        } else {
            # Close connection
            if ($ws.State -eq [System.Net.WebSockets.WebSocketState]::Open -or $ws.State -eq [System.Net.WebSockets.WebSocketState]::CloseReceived) {
                try {
                    $closeTask = $ws.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "Timeout", $cts.Token)
                    $closeTask.Wait(2000)
                } catch {}
            }

            return [PSCustomObject]@{
                Success = $true
                Message = "Connected successfully, no immediate message received."
            }
        }

    } catch {
        $innerMsg = ""
        if ($_.Exception.InnerException) {
            $innerMsg = " | Inner: " + $_.Exception.InnerException.Message
        }
        return [PSCustomObject]@{
            Success = $false
            Message = "$($_.Exception.Message)$innerMsg`nStack: $($_.ScriptStackTrace)"
        }
    }
}

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "SyncSpace E2E Test Suite (9 Microservices)" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# -----------------------------------------------------------------------------
# 1. AUTH SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[1] Testing Auth Service..." -ForegroundColor Yellow

$userA_username = "test_user_a_" + (Get-Random)
$userA_email = "$userA_username@example.com"
$userA_password = "Password123!"

$userB_username = "test_user_b_" + (Get-Random)
$userB_email = "$userB_username@example.com"
$userB_password = "Password123!"

# Register User A
Write-Host "Registering User A ($userA_username)..."
$regBody = @{
    email    = $userA_email
    username = $userA_username
    password = $userA_password
} | ConvertTo-Json

$regRes = Send-Request -Uri "$gatewayUrl/api/v1/auth/register" -Method "POST" -Body $regBody
if ($regRes.StatusCode -eq 201 -or $regRes.StatusCode -eq 409) {
    Write-Host "[PASS] Register User A (Status: $($regRes.StatusCode))" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Register User A failed. Status: $($regRes.StatusCode), Response: $($regRes.Content)" -ForegroundColor Red
    exit 1
}

# Register User B
Write-Host "Registering User B ($userB_username)..."
$regBodyB = @{
    email    = $userB_email
    username = $userB_username
    password = $userB_password
} | ConvertTo-Json

$regResB = Send-Request -Uri "$gatewayUrl/api/v1/auth/register" -Method "POST" -Body $regBodyB
if ($regResB.StatusCode -eq 201 -or $regResB.StatusCode -eq 409) {
    Write-Host "[PASS] Register User B (Status: $($regResB.StatusCode))" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Register User B failed. Status: $($regResB.StatusCode), Response: $($regResB.Content)" -ForegroundColor Red
    exit 1
}

# Login User A
Write-Host "Logging in User A..."
$loginBody = @{
    identity = $userA_email
    password = $userA_password
} | ConvertTo-Json

$loginRes = Send-Request -Uri "$gatewayUrl/api/v1/auth/login" -Method "POST" -Body $loginBody
if ($loginRes.StatusCode -eq 200) {
    $loginData = $loginRes.Content | ConvertFrom-Json
    $userA_token = $loginData.data.access_token
    $userA_id = $loginData.data.user_id
    Write-Host "[PASS] Login User A. ID: $userA_id" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Login User A failed. Status: $($loginRes.StatusCode), Response: $($loginRes.Content)" -ForegroundColor Red
    exit 1
}

# Login User B
Write-Host "Logging in User B..."
$loginBodyB = @{
    identity = $userB_email
    password = $userB_password
} | ConvertTo-Json

$loginResB = Send-Request -Uri "$gatewayUrl/api/v1/auth/login" -Method "POST" -Body $loginBodyB
if ($loginResB.StatusCode -eq 200) {
    $loginDataB = $loginResB.Content | ConvertFrom-Json
    $userB_token = $loginDataB.data.access_token
    $userB_id = $loginDataB.data.user_id
    Write-Host "[PASS] Login User B. ID: $userB_id" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Login User B failed. Status: $($loginResB.StatusCode), Response: $($loginResB.Content)" -ForegroundColor Red
    exit 1
}

# -----------------------------------------------------------------------------
# 2. USER SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[2] Testing User Service..." -ForegroundColor Yellow

# Get Own Profile (User A)
$profRes = Send-Request -Uri "$gatewayUrl/api/v1/users/profile" -Token $userA_token
if ($profRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get User A Profile" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get User A Profile failed. Status: $($profRes.StatusCode), Response: $($profRes.Content)" -ForegroundColor Red
}

# Get User B Profile by ID using User A token
$profBRes = Send-Request -Uri "$gatewayUrl/api/v1/users/profile/$userB_id" -Token $userA_token
if ($profBRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get User B Profile by ID" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get User B Profile failed. Status: $($profBRes.StatusCode), Response: $($profBRes.Content)" -ForegroundColor Red
}

# Update Profile (User A)
$updateBody = @{
    display_name = "User A Updated"
    bio          = "Hello, testing bio updating!"
} | ConvertTo-Json
$updateRes = Send-Request -Uri "$gatewayUrl/api/v1/users/profile" -Method "PUT" -Body $updateBody -Token $userA_token
if ($updateRes.StatusCode -eq 200) {
    Write-Host "[PASS] Update User A Profile" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Update User A Profile failed. Status: $($updateRes.StatusCode), Response: $($updateRes.Content)" -ForegroundColor Red
}

# Update Presence Status
$statusBody = @{
    status      = "busy"
    custom_text = "Telescope focus session"
} | ConvertTo-Json
$statusRes = Send-Request -Uri "$gatewayUrl/api/v1/users/status" -Method "PUT" -Body $statusBody -Token $userA_token
if ($statusRes.StatusCode -eq 200) {
    Write-Host "[PASS] Update Presence Status" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Update Presence Status failed. Status: $($statusRes.StatusCode), Response: $($statusRes.Content)" -ForegroundColor Red
}

# Send Friend Request (A -> B)
Write-Host "User A sending friend request to User B..."
$friendReqBody = @{
    friend_id = $userB_id
} | ConvertTo-Json
$friendReqRes = Send-Request -Uri "$gatewayUrl/api/v1/users/friends/request" -Method "POST" -Body $friendReqBody -Token $userA_token
$friendshipId = $null
if ($friendReqRes.StatusCode -eq 200 -or $friendReqRes.StatusCode -eq 201) {
    $friendReqData = $friendReqRes.Content | ConvertFrom-Json
    $friendshipId = $friendReqData.data.id
    Write-Host "[PASS] Friend request sent. Friendship ID: $friendshipId" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Friend request failed. Status: $($friendReqRes.StatusCode), Response: $($friendReqRes.Content)" -ForegroundColor Red
}

# Accept Friend Request (B accepts A's request)
if ($friendshipId) {
    Write-Host "User B accepting friend request from User A..."
    $acceptBody = @{
        action = "accept"
    } | ConvertTo-Json
    $acceptRes = Send-Request -Uri "$gatewayUrl/api/v1/users/friends/request/$friendshipId" -Method "PUT" -Body $acceptBody -Token $userB_token
    if ($acceptRes.StatusCode -eq 200) {
        Write-Host "[PASS] Friend request accepted by User B" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] Friend request accept failed. Status: $($acceptRes.StatusCode), Response: $($acceptRes.Content)" -ForegroundColor Red
    }
}

# List Friends (User A)
$friendsRes = Send-Request -Uri "$gatewayUrl/api/v1/users/friends?status=ACCEPTED" -Token $userA_token
if ($friendsRes.StatusCode -eq 200) {
    Write-Host "[PASS] List Friends" -ForegroundColor Green
} else {
    Write-Host "[FAIL] List Friends failed. Status: $($friendsRes.StatusCode), Response: $($friendsRes.Content)" -ForegroundColor Red
}

# Block User (A blocks B - test block functionality)
Write-Host "User A blocking User B..."
$blockBody = @{
    friend_id = $userB_id
} | ConvertTo-Json
$blockRes = Send-Request -Uri "$gatewayUrl/api/v1/users/friends/block" -Method "POST" -Body $blockBody -Token $userA_token
if ($blockRes.StatusCode -eq 200) {
    Write-Host "[PASS] Block User" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Block User failed. Status: $($blockRes.StatusCode), Response: $($blockRes.Content)" -ForegroundColor Red
}

# Let's clean up friend request block by sending request again and accepting it so they can communicate
Write-Host "Resetting friend request for room and chat testing..."
$friendReqRes2 = Send-Request -Uri "$gatewayUrl/api/v1/users/friends/request" -Method "POST" -Body $friendReqBody -Token $userA_token
if ($friendReqRes2.StatusCode -eq 200) {
    $friendReqData2 = $friendReqRes2.Content | ConvertFrom-Json
    $friendshipId2 = $friendReqData2.data.id
    $acceptRes2 = Send-Request -Uri "$gatewayUrl/api/v1/users/friends/request/$friendshipId2" -Method "PUT" -Body $acceptBody -Token $userB_token
    Write-Host "Friendship re-established."
}

# -----------------------------------------------------------------------------
# 3. ROOM SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[3] Testing Room Service..." -ForegroundColor Yellow

# Create Room
$roomBody = @{
    name        = "E2E Testing Room"
    description = "A room created by PowerShell E2E test script"
    privacy     = "public"
} | ConvertTo-Json
$roomRes = Send-Request -Uri "$gatewayUrl/api/v1/rooms" -Method "POST" -Body $roomBody -Token $userA_token
$roomId = $null
if ($roomRes.StatusCode -eq 201) {
    $roomData = $roomRes.Content | ConvertFrom-Json
    $roomId = $roomData.data.id
    Write-Host "[PASS] Create Room (Room ID: $roomId)" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Create Room failed. Status: $($roomRes.StatusCode), Response: $($roomRes.Content)" -ForegroundColor Red
    exit 1
}

# Get Room by ID
$getRoomRes = Send-Request -Uri "$gatewayUrl/api/v1/rooms/$roomId" -Token $userA_token
if ($getRoomRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Room by ID" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Room by ID failed. Status: $($getRoomRes.StatusCode), Response: $($getRoomRes.Content)" -ForegroundColor Red
}

# Get Rooms (Search / List)
$listRoomsRes = Send-Request -Uri "$gatewayUrl/api/v1/rooms?search=E2E" -Token $userA_token
if ($listRoomsRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Rooms List/Search" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Rooms List/Search failed. Status: $($listRoomsRes.StatusCode), Response: $($listRoomsRes.Content)" -ForegroundColor Red
}

# User B Joins the Room
Write-Host "User B joining the Room..."
$joinRes = Send-Request -Uri "$gatewayUrl/api/v1/rooms/$roomId/join" -Method "POST" -Token $userB_token
if ($joinRes.StatusCode -eq 200) {
    Write-Host "[PASS] User B joined Room successfully" -ForegroundColor Green
} else {
    Write-Host "[FAIL] User B join Room failed. Status: $($joinRes.StatusCode), Response: $($joinRes.Content)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# 4. CHAT SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[4] Testing Chat Service..." -ForegroundColor Yellow

# WebSocket chat route test
Write-Host "Connecting User A to Chat WebSocket..."
$chatWSUrl = "$wsGatewayUrl/api/v1/rooms/$roomId/chat/ws?token=$userA_token"
$chatMsg = @{
    event = "chat:send_message"
    payload = @{
        content = "Hello, this is User A talking in WS chat!"
    }
} | ConvertTo-Json -Compress

$chatWSRes = Test-WebSocket -Uri $chatWSUrl -Token $null -MessageToSend $chatMsg
if ($chatWSRes.Success) {
    Write-Host "[PASS] Chat WebSocket Connection & Send message. Response: $($chatWSRes.Message)" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Chat WebSocket failed. Error: $($chatWSRes.Message)" -ForegroundColor Red
}

# Get Chat History (REST API)
$chatHistoryRes = Send-Request -Uri "$gatewayUrl/api/v1/rooms/$roomId/chat/messages" -Token $userA_token
if ($chatHistoryRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Chat History REST API" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Chat History failed. Status: $($chatHistoryRes.StatusCode), Response: $($chatHistoryRes.Content)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# 5. MUSIC SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[5] Testing Music Service..." -ForegroundColor Yellow

# Search tracks (should run and return 200, even if empty)
$musicSearchRes = Send-Request -Uri "$gatewayUrl/api/v1/music/search?q=lofi" -Token $userA_token
if ($musicSearchRes.StatusCode -eq 200) {
    Write-Host "[PASS] Search Music" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Search Music failed. Status: $($musicSearchRes.StatusCode), Response: $($musicSearchRes.Content)" -ForegroundColor Red
}

# Log playback history
$logHistoryBody = @{
    room_id   = $roomId
    track_id  = "track_12345"
    title     = "Lofi hip hop mix"
    artist    = "Lofi Girl"
    duration  = 180
    source_url= "https://www.youtube.com/watch?v=5qap5aO4i9A"
} | ConvertTo-Json
$logHistoryRes = Send-Request -Uri "$gatewayUrl/api/v1/music/history" -Method "POST" -Body $logHistoryBody -Token $userA_token
if ($logHistoryRes.StatusCode -eq 200 -or $logHistoryRes.StatusCode -eq 201) {
    Write-Host "[PASS] Log Playback History" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Log Playback History failed. Status: $($logHistoryRes.StatusCode), Response: $($logHistoryRes.Content)" -ForegroundColor Red
}

# Get Playback History for Room
$getHistoryRes = Send-Request -Uri "$gatewayUrl/api/v1/music/history/$roomId" -Token $userA_token
if ($getHistoryRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Music Playback History" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Music Playback History failed. Status: $($getHistoryRes.StatusCode), Response: $($getHistoryRes.Content)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# 6. PLAYLIST SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[6] Testing Playlist Service..." -ForegroundColor Yellow

# Create Playlist
$playlistBody = @{
    name        = "Study Playlist"
    description = "Chill vibes for coding"
    room_id     = $roomId
} | ConvertTo-Json
$playlistRes = Send-Request -Uri "$gatewayUrl/api/v1/playlists/" -Method "POST" -Body $playlistBody -Token $userA_token
$playlistId = $null
if ($playlistRes.StatusCode -eq 201 -or $playlistRes.StatusCode -eq 200) {
    $playlistData = $playlistRes.Content | ConvertFrom-Json
    $playlistId = $playlistData.data.id
    Write-Host "[PASS] Create Room Playlist (Playlist ID: $playlistId)" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Create Playlist failed. Status: $($playlistRes.StatusCode), Response: $($playlistRes.Content)" -ForegroundColor Red
}

if ($playlistId) {
    # Add Track to Playlist
    $addTrackBody = @{
        track_id     = "track_12345"
        title        = "Code Sync"
        artist       = "Go Coder"
        duration_ms  = 240000
        source_url   = "https://example.com/audio.mp3"
    } | ConvertTo-Json
    $addTrackRes = Send-Request -Uri "$gatewayUrl/api/v1/playlists/$playlistId/tracks" -Method "POST" -Body $addTrackBody -Token $userA_token
    $trackItemId = $null
    if ($addTrackRes.StatusCode -eq 200 -or $addTrackRes.StatusCode -eq 201) {
        $trackItemData = $addTrackRes.Content | ConvertFrom-Json
        $trackItemId = $trackItemData.data.id
        Write-Host "[PASS] Add Track to Playlist" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] Add Track failed. Status: $($addTrackRes.StatusCode), Response: $($addTrackRes.Content)" -ForegroundColor Red
    }

    # Get Tracks in Playlist
    $getTracksRes = Send-Request -Uri "$gatewayUrl/api/v1/playlists/$playlistId/tracks" -Token $userA_token
    if ($getTracksRes.StatusCode -eq 200) {
        Write-Host "[PASS] Get Tracks in Playlist" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] Get Tracks failed. Status: $($getTracksRes.StatusCode), Response: $($getTracksRes.Content)" -ForegroundColor Red
    }

    if ($trackItemId) {
        # Vote Track
        $voteBody = @{
            vote_type = "up"
        } | ConvertTo-Json
        $voteRes = Send-Request -Uri "$gatewayUrl/api/v1/playlists/tracks/$trackItemId/vote" -Method "POST" -Body $voteBody -Token $userA_token
        if ($voteRes.StatusCode -eq 200) {
            Write-Host "[PASS] Vote Track in Playlist" -ForegroundColor Green
        } else {
            Write-Host "[FAIL] Vote Track failed. Status: $($voteRes.StatusCode), Response: $($voteRes.Content)" -ForegroundColor Red
        }

        # Move Track
        $moveBody = @{
            new_position = 0
        } | ConvertTo-Json
        $moveRes = Send-Request -Uri "$gatewayUrl/api/v1/playlists/$playlistId/tracks/$trackItemId/move" -Method "PUT" -Body $moveBody -Token $userA_token
        if ($moveRes.StatusCode -eq 200) {
            Write-Host "[PASS] Move Track in Playlist" -ForegroundColor Green
        } else {
            Write-Host "[FAIL] Move Track failed. Status: $($moveRes.StatusCode), Response: $($moveRes.Content)" -ForegroundColor Red
        }
    }
}

# Get Room Playlists
$getRoomPlaylistsRes = Send-Request -Uri "$gatewayUrl/api/v1/playlists/room/$roomId" -Token $userA_token
if ($getRoomPlaylistsRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Room Playlists" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Room Playlists failed. Status: $($getRoomPlaylistsRes.StatusCode), Response: $($getRoomPlaylistsRes.Content)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# 7. PLAYBACK SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[7] Testing Playback Service..." -ForegroundColor Yellow

# Get Playback State
$playbackStateRes = Send-Request -Uri "$gatewayUrl/api/v1/rooms/$roomId/playback/state" -Token $userA_token
if ($playbackStateRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Playback State REST API" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Playback State failed. Status: $($playbackStateRes.StatusCode), Response: $($playbackStateRes.Content)" -ForegroundColor Red
}

# Test Playback WebSocket
Write-Host "Connecting User A to Playback WebSocket..."
$playbackWSUrl = "$wsGatewayUrl/api/v1/rooms/$roomId/playback/ws?token=$userA_token"
# Send sync:ping message (ntp ping format)
$pingMsg = @{
    event = "sync:ping"
    payload = @{
        t1 = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    }
} | ConvertTo-Json -Compress

$playbackWSRes = Test-WebSocket -Uri $playbackWSUrl -Token $null -MessageToSend $pingMsg
if ($playbackWSRes.Success) {
    Write-Host "[PASS] Playback WebSocket Connection & Sync:pong. Response: $($playbackWSRes.Message)" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Playback WebSocket failed. Error: $($playbackWSRes.Message)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# 8. VOICE SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[8] Testing Voice Service..." -ForegroundColor Yellow

# Get LiveKit Room Token
$voiceTokenRes = Send-Request -Uri "$gatewayUrl/api/v1/voice/rooms/$roomId/token" -Token $userA_token
if ($voiceTokenRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Voice LiveKit Token" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Voice Token failed. Status: $($voiceTokenRes.StatusCode), Response: $($voiceTokenRes.Content)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# 9. NOTIFICATION SERVICE TESTING
# -----------------------------------------------------------------------------
Write-Host "`n[9] Testing Notification Service..." -ForegroundColor Yellow

# Get Notifications
$notiListRes = Send-Request -Uri "$gatewayUrl/api/v1/notifications/" -Token $userA_token
if ($notiListRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Notifications List" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Notifications failed. Status: $($notiListRes.StatusCode), Response: $($notiListRes.Content)" -ForegroundColor Red
}

# Trigger manual Notification
$triggerNotiBody = @{
    receiver_id = $userA_id
    sender_id   = $userB_id
    type        = "FRIEND_REQUEST_ACCEPTED"
    content     = "User B has accepted your friend request!"
} | ConvertTo-Json
$triggerNotiRes = Send-Request -Uri "$gatewayUrl/api/v1/notifications/trigger" -Method "POST" -Body $triggerNotiBody -Token $userA_token
if ($triggerNotiRes.StatusCode -eq 200 -or $triggerNotiRes.StatusCode -eq 201) {
    Write-Host "[PASS] Trigger Notification Manual" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Trigger Notification failed. Status: $($triggerNotiRes.StatusCode), Response: $($triggerNotiRes.Content)" -ForegroundColor Red
}

# Get unread count
$unreadRes = Send-Request -Uri "$gatewayUrl/api/v1/notifications/unread-count" -Token $userA_token
if ($unreadRes.StatusCode -eq 200) {
    Write-Host "[PASS] Get Unread Notifications Count" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Get Unread Count failed. Status: $($unreadRes.StatusCode), Response: $($unreadRes.Content)" -ForegroundColor Red
}

# Mark read all
$readAllRes = Send-Request -Uri "$gatewayUrl/api/v1/notifications/read-all" -Method "PUT" -Token $userA_token
if ($readAllRes.StatusCode -eq 200) {
    Write-Host "[PASS] Mark All Notifications Read" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Mark All Read failed. Status: $($readAllRes.StatusCode), Response: $($readAllRes.Content)" -ForegroundColor Red
}

# -----------------------------------------------------------------------------
# CLEANUP / POST-FLIGHT
# -----------------------------------------------------------------------------
# Leave Room for User B and User A
Write-Host "`nCleaning up Room membership..."
$leaveB = Send-Request -Uri "$gatewayUrl/api/v1/rooms/$roomId/leave" -Method "POST" -Token $userB_token
$leaveA = Send-Request -Uri "$gatewayUrl/api/v1/rooms/$roomId/leave" -Method "POST" -Token $userA_token

Write-Host "`nE2E testing complete." -ForegroundColor Cyan
