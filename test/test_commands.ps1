# Test health endpoint
Write-Host "Testing health endpoint..."
Invoke-WebRequest -Uri "http://localhost:8080/health" -Method Get | Select-Object -ExpandProperty Content

# Test log analysis
Write-Host "`nTesting log analysis..."
$headers = @{
    "Content-Type" = "application/json"
}
$body = Get-Content -Raw "test/sample_logs.json"
Invoke-WebRequest -Uri "http://localhost:8080/analyze/logs" -Method Post -Headers $headers -Body $body | Select-Object -ExpandProperty Content

# Test performance analysis
Write-Host "`nTesting performance analysis..."
Invoke-WebRequest -Uri "http://localhost:8080/analyze/performance" -Method Post -Headers $headers -Body $body | Select-Object -ExpandProperty Content

# Test CSV conversion
Write-Host "`nTesting CSV conversion..."
$csvResponse = Invoke-WebRequest -Uri "http://localhost:8080/convert/to-csv" -Method Post -Headers $headers -Body $body
$csvResponse.Headers.'Content-Disposition'
$csvResponse.Content 