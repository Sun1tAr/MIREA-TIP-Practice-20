# generate-load.ps1
# Script for generating test load for Tasks service metrics

Write-Host "Generating load for metrics..." -ForegroundColor Green

Write-Host "`n1. Successful GET requests (50 times)" -ForegroundColor Yellow
for ($i=1; $i -le 50; $i++) {
    curl.exe -s -X GET http://localhost:8082/v1/tasks -H "Authorization: Bearer demo-token" | Out-Null
    Write-Host -NoNewline "."
}
Write-Host " OK"

Write-Host "2. Creating tasks (20 times)" -ForegroundColor Yellow
for ($i=1; $i -le 20; $i++) {
    $body = "{`"title`":`"Test $i`",`"description`":`"Load test`"}"
    curl.exe -s -X POST http://localhost:8082/v1/tasks `
        -H "Content-Type: application/json" `
        -H "Authorization: Bearer demo-token" `
        -d $body | Out-Null
    Write-Host -NoNewline "."
}
Write-Host " OK"

Write-Host "3. Authorization errors (30 times)" -ForegroundColor Yellow
for ($i=1; $i -le 30; $i++) {
    curl.exe -s -X GET http://localhost:8082/v1/tasks -H "Authorization: Bearer wrong-token" | Out-Null
    Write-Host -NoNewline "."
}
Write-Host " OK"

Write-Host "4. Requests to non-existent tasks (10 times)" -ForegroundColor Yellow
for ($i=1; $i -le 10; $i++) {
    curl.exe -s -X GET http://localhost:8082/v1/tasks/t_non-existent -H "Authorization: Bearer demo-token" | Out-Null
    Write-Host -NoNewline "."
}
Write-Host " OK"

Write-Host "`nLoad generation completed!" -ForegroundColor Green