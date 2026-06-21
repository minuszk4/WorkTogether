#!/usr/bin/env pwsh
# migrate.ps1 - Chạy tất cả database migrations cho WorkTogether
# Usage: .\migrate.ps1 [up|down]

param(
    [string]$Action = "up"
)

Write-Host "=== WorkTogether Database Migration ($Action) ===" -ForegroundColor Cyan

$services = @(
    @{ Name = "auth-service";    DB = "worktogether_auth" },
    @{ Name = "user-service";    DB = "worktogether_user" },
    @{ Name = "room-service";    DB = "worktogether_room" },
    @{ Name = "chat-service";    DB = "worktogether_chat" },
    @{ Name = "collab-service";  DB = "worktogether_collab" }
)


foreach ($svc in $services) {
    $migDir = "services/$($svc.Name)/db/migrations"
    
    if (-not (Test-Path $migDir)) {
        Write-Host "  [SKIP] $($svc.Name) - không có thư mục migrations" -ForegroundColor Yellow
        continue
    }

    $pattern = if ($Action -eq "up") { "*.up.sql" } else { "*.down.sql" }
    $files = Get-ChildItem -Path $migDir -Filter $pattern | Sort-Object Name

    foreach ($file in $files) {
        $sqlContent = Get-Content $file.FullName -Raw
        Write-Host "  [RUN] $($svc.DB) <- $($file.Name)" -ForegroundColor Green
        $sqlContent | docker exec -i worktogether_postgres psql -U postgres -d $svc.DB
    }

    Write-Host "  [OK] $($svc.Name) migration complete" -ForegroundColor Green
}

Write-Host ""
Write-Host "=== Migration hoàn tất ===" -ForegroundColor Cyan
Write-Host "Kiểm tra bảng trong auth DB:" -ForegroundColor Gray
docker exec worktogether_postgres psql -U postgres -d worktogether_auth -c "\dt"
