# Set working directory to the script's location
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -Path $scriptPath

# Read sample logs once and reuse
$logData = Get-Content -Raw "sample_logs.json"
if (-not $logData) {
    Write-Host "Failed to read sample_logs.json"
    exit 1
}

# Common headers
$headers = @{"Content-Type" = "application/json"}

# Test health endpoint (fast, no body needed)
Write-Host "Testing health endpoint..."
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8081/health" -Method Get -TimeoutSec 5
    Write-Host "Health check successful: $($response.Content)"
} catch {
    Write-Host "Health check failed: $($_.Exception.Message)"
}

# Test log analysis endpoint
Write-Host "`nTesting log analysis endpoint..."
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8081/analyze/logs" -Method Post -Headers $headers -Body $logData -TimeoutSec 10
    Write-Host "Log analysis successful:`n$($response.Content)"
} catch {
    Write-Host "Log analysis failed: $($_.Exception.Message)"
}

# Test performance analysis endpoint
Write-Host "`nTesting performance analysis endpoint..."
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8081/analyze/performance" -Method Post -Headers $headers -Body $logData -TimeoutSec 10
    Write-Host "Performance analysis successful:`n$($response.Content)"
} catch {
    Write-Host "Performance analysis failed: $($_.Exception.Message)"
}

# Test CSV conversion endpoint (fast, no API call)
Write-Host "`nTesting CSV conversion endpoint..."
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8081/convert/to-csv" -Method Post -Headers $headers -Body $logData -TimeoutSec 5
    Write-Host "CSV conversion successful. Filename: $($response.Headers['Content-Disposition'])"
    
    # Convert the response content to bytes
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($response.Content)
    $outputPath = Join-Path $scriptPath "output.csv"
    [System.IO.File]::WriteAllBytes($outputPath, $bytes)
    
    Write-Host "`nCSV content saved to $outputPath"
    Write-Host "`nFirst few lines of CSV content:"
    Get-Content $outputPath -TotalCount 5 -Encoding UTF8 | ForEach-Object { Write-Host $_ }
} catch {
    Write-Host "CSV conversion failed: $($_.Exception.Message)"
}

# Test file upload endpoint
Write-Host "`nTesting file upload endpoint..."
try {
    $boundary = [System.Guid]::NewGuid().ToString()
    $LF = "`r`n"
    
    $bodyLines = (
        "--$boundary",
        "Content-Disposition: form-data; name=`"file`"; filename=`"sample_logs.json`"",
        "Content-Type: application/json$LF",
        $logData,  # Reuse the already loaded log data
        "--$boundary--$LF"
    ) -join $LF

    $response = Invoke-WebRequest -Uri "http://localhost:8081/upload" -Method Post `
        -ContentType "multipart/form-data; boundary=$boundary" `
        -Body $bodyLines `
        -TimeoutSec 10
    
    Write-Host "File upload successful:`n$($response.Content)"
} catch {
    Write-Host "File upload failed: $($_.Exception.Message)"
}