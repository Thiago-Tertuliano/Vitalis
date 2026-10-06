# Novo microserviço a partir do template (PowerShell)
# Uso:
#   .\scripts\new-service.ps1 -Name Vitalis-billing -Module github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-billing

param(
  [Parameter(Mandatory = $true)][string]$Name,
  [Parameter(Mandatory = $true)][string]$Module
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Dest = Join-Path (Resolve-Path (Join-Path $Root "..\..")) $Name

if (Test-Path $Dest) { throw "Destino já existe: $Dest" }

New-Item -ItemType Directory -Path $Dest | Out-Null
Copy-Item -Path (Join-Path $Root "*") -Destination $Dest -Recurse -Force
Remove-Item -Recurse -Force (Join-Path $Dest ".git") -ErrorAction SilentlyContinue

$oldModule = "github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template"
Get-ChildItem -Path $Dest -Recurse -Include *.go,go.mod,README.md,.env.example,Makefile |
  ForEach-Object {
    $c = Get-Content $_.FullName -Raw -ErrorAction SilentlyContinue
    if ($null -eq $c) { return }
    $c = $c.Replace($oldModule, $Module).Replace("vitalis-service-template", $Name)
    Set-Content -Path $_.FullName -Value $c -Encoding UTF8
  }

Write-Host "Criado: $Dest"
Write-Host "Próximos: cd $Dest; copy .env.example .env; go mod tidy; docker compose up --build"
