param(
    [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"

function Assert-Equal($Actual, $Expected, $Message) {
    if ($Actual -ne $Expected) {
        throw "ASSERTION FAILED: $Message (expected '$Expected', got '$Actual')"
    }
}

Write-Host "1. Checking /health..." -ForegroundColor Cyan
$health = Invoke-RestMethod -Method Get -Uri "$BaseUrl/health"
Assert-Equal $health.success $true "Health endpoint should return success=true"

$email = "tester_$(Get-Random)@example.com"
Write-Host "2. Signing up user ($email)..." -ForegroundColor Cyan
$signupBody = @{
    name     = "Tester Ayam Batu"
    email    = $email
    password = "password123"
} | ConvertTo-Json
$signup = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/auth/signup" -ContentType "application/json" -Body $signupBody
$token = $signup.data.token
if (-not $token) { throw "Sign up did not return JWT token" }
$headers = @{ Authorization = "Bearer $token" }

Write-Host "3. Testing login..." -ForegroundColor Cyan
$loginBody = @{ email = $email; password = "password123" } | ConvertTo-Json
$login = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/auth/login" -ContentType "application/json" -Body $loginBody
Assert-Equal $login.success $true "Login should succeed"

Write-Host "4. Creating product..." -ForegroundColor Cyan
$prodBody = @{
    name        = "Ayam Batu Kampung"
    sku         = "ABK-01"
    unit        = "ekor"
    description = "Produk uji coba"
} | ConvertTo-Json
$prod = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/products" -Headers $headers -ContentType "application/json" -Body $prodBody
$productId = $prod.data.id

Write-Host "5. Adding Batch 1 (10 @ 20000) & Batch 2 (10 @ 30000)..." -ForegroundColor Cyan
$b1 = @{ product_id = $productId; batch_code = "B1"; buy_price = 20000; initial_stock = 10; purchased_at = "2026-11-25" } | ConvertTo-Json
$b2 = @{ product_id = $productId; batch_code = "B2"; buy_price = 30000; initial_stock = 10; purchased_at = "2026-11-26" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/batches" -Headers $headers -ContentType "application/json" -Body $b1 | Out-Null
Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/batches" -Headers $headers -ContentType "application/json" -Body $b2 | Out-Null

$detail = Invoke-RestMethod -Method Get -Uri "$BaseUrl/api/v1/products/$productId" -Headers $headers
Assert-Equal $detail.data.total_bought_stock 20 "Total bought stock should be 20"
Assert-Equal $detail.data.min_sell_price 25000 "Min sell price (simple average of 20000 and 30000) should be 25000"

Write-Host "6. Verifying sell_price below min_sell_price (24000 < 25000) is rejected..." -ForegroundColor Cyan
$lowSale = @{ product_id = $productId; quantity = 2; sell_price = 24000; sold_at = "2026-11-26" } | ConvertTo-Json
$rejected = $false
try {
    Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/sales" -Headers $headers -ContentType "application/json" -Body $lowSale | Out-Null
} catch {
    if ($_.Exception.Response.StatusCode.value__ -eq 422) {
        $rejected = $true
    } else {
        throw
    }
}
Assert-Equal $rejected $true "Sale below min_sell_price must return HTTP 422"

Write-Host "7. Recording valid daily sale (12 @ 35000 -> FIFO 10x20000 + 2x30000 = 260000 modal)..." -ForegroundColor Cyan
$validSale = @{ product_id = $productId; quantity = 12; sell_price = 35000; sold_at = "2026-11-26"; notes = "Penjualan hari ini" } | ConvertTo-Json
$sale = Invoke-RestMethod -Method Post -Uri "$BaseUrl/api/v1/sales" -Headers $headers -ContentType "application/json" -Body $validSale
$saleId = $sale.data.id
Assert-Equal $sale.data.total_modal 260000 "Modal should be 260000 (FIFO)"
Assert-Equal $sale.data.total_bruto 420000 "Bruto should be 420000"
Assert-Equal $sale.data.total_netto 160000 "Netto should be 160000"

Write-Host "8. Checking financial report & remaining stock..." -ForegroundColor Cyan
$report = Invoke-RestMethod -Method Get -Uri "$BaseUrl/api/v1/reports/financial" -Headers $headers
Assert-Equal $report.data.modal 260000 "Report modal should be 260000"
Assert-Equal $report.data.bruto 420000 "Report bruto should be 420000"
Assert-Equal $report.data.netto 160000 "Report netto should be 160000"
Assert-Equal $report.data.remaining_stock 8 "Remaining stock should be 8"

Write-Host "9. Deleting sale and verifying stock is restored to 20..." -ForegroundColor Cyan
Invoke-RestMethod -Method Delete -Uri "$BaseUrl/api/v1/sales/$saleId" -Headers $headers | Out-Null
$afterDelete = Invoke-RestMethod -Method Get -Uri "$BaseUrl/api/v1/products/$productId" -Headers $headers
Assert-Equal $afterDelete.data.remaining_stock 20 "Remaining stock should restore to 20 after deleting sale"

Write-Host "`nALL END-TO-END API CHECKS PASSED!" -ForegroundColor Green
