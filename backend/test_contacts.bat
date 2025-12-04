@echo off
echo Testing GetContacts API...
echo.

REM First, login to get a token
echo Step 1: Logging in as alice.author@test.com...
curl -X POST http://localhost:8080/api/v1/auth/login ^
  -H "Content-Type: application/json" ^
  -d "{\"email\":\"alice.author@test.com\",\"password\":\"password123\"}" ^
  -o login_response.json

echo.
echo Login response saved to login_response.json
type login_response.json
echo.
echo.

REM Extract token (you'll need to manually copy it)
echo Step 2: Copy the token from above and run:
echo curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/v1/chat/contacts
echo.

pause
