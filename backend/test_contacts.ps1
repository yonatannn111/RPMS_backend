$baseUrl = "http://localhost:8080"

Write-Host "=== Testing Chat Contacts API ===" -ForegroundColor Cyan
Write-Host ""

# Login
Write-Host "1. Logging in as alice.author@test.com..." -ForegroundColor Yellow
$loginBody = @{
    email = "alice.author@test.com"
    password = "password123"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
    Write-Host "   ✓ Login successful!" -ForegroundColor Green
    Write-Host "   User: $($loginResponse.user.name)" -ForegroundColor Gray
    Write-Host "   Role: $($loginResponse.user.role)" -ForegroundColor Gray
    Write-Host "   Token: $($loginResponse.token.Substring(0, 20))..." -ForegroundColor Gray
    
    # Get Contacts
    Write-Host ""
    Write-Host "2. Fetching contacts..." -ForegroundColor Yellow
    $headers = @{
        "Authorization" = "Bearer $($loginResponse.token)"
    }
    
    $contacts = Invoke-RestMethod -Uri "$baseUrl/api/v1/chat/contacts" -Method Get -Headers $headers
    
    Write-Host "   Response type: $($contacts.GetType().Name)" -ForegroundColor Gray
    Write-Host "   Contacts returned: $($contacts.Count)" -ForegroundColor Gray
    
    if ($contacts.Count -gt 0) {
        Write-Host ""
        Write-Host "   Contact List:" -ForegroundColor Green
        foreach ($contact in $contacts) {
            Write-Host "     - $($contact.name) ($($contact.role)) - $($contact.email)" -ForegroundColor White
        }
    } else {
        Write-Host "   ⚠ NO CONTACTS FOUND!" -ForegroundColor Red
        Write-Host "   Raw response: $($contacts | ConvertTo-Json)" -ForegroundColor Gray
    }
    
} catch {
    Write-Host "   ✗ Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "   Response: $($_.ErrorDetails.Message)" -ForegroundColor Gray
}

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan
