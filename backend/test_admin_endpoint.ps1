#!/usr/bin/env pwsh

# Test the admin user endpoint
$token = Get-Content -Path "test_token.txt" -ErrorAction SilentlyContinue

if (-not $token) {
    Write-Host "No token found. Please login first."
    exit 1
}

Write-Host "Testing /api/v1/users/admin endpoint..."
$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/users/admin" `
    -Headers @{"Authorization" = "Bearer $token"} `
    -Method GET `
    -ErrorAction SilentlyContinue

if ($response) {
    Write-Host "Status: $($response.StatusCode)"
    Write-Host "Response:"
    $response.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10
} else {
    Write-Host "Request failed"
}
