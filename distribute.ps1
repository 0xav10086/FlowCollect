# warn: 此脚本已过时,202501140936
# distribute.ps1 - FlowCollect 多平台自动化编译分发脚本
# 编译输出到: FlowCollect\dist\<版本>\<平台>\
# 版本格式: 年月日小时 (如: 2024121514)

# 设置默认编码为 UTF-8
$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'

Write-Host "=== FlowCollect 自动化分发工具 ===" -ForegroundColor Cyan
Write-Host "编译输出目录: FlowCollect\dist\<版本>\<平台>\" -ForegroundColor Yellow

# 0. 准备版本和目录
$version = Get-Date -Format "yyyyMMddHH"  # 年月日小时，如 2024121514
$tagName = "v$version"
$baseDir = ".\dist\$version"
$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

Write-Host "当前版本: $version ($timestamp)" -ForegroundColor Green

# 1. 收集用户输入（带默认值）
$targetID = Read-Host "请输入设备 ID [例如: OpenWrt-VM / Android-Phone]"
if ($targetID -eq "") { $targetID = "86" }

$apiAddr = Read-Host "请输入 Mihomo API 地址 [默认: http://127.0.0.1:9097]"
if ($apiAddr -eq "") { $apiAddr = "http://127.0.0.1:9097" }

$secret = Read-Host "请输入 Mihomo Secret [默认: abcd]"
if ($secret -eq "") { $secret = "abcd" }

Write-Host "`n请选择目标平台:" -ForegroundColor Yellow
Write-Host "1. Windows (AMD64) - .exe"
Write-Host "2. Android (ARM64) - 手机通用"
Write-Host "3. OpenWrt (x86_64) - 群晖虚拟机"
Write-Host "4. OpenWrt (MIPSLE) - 硬路由"
Write-Host "5. Linux (AMD64) - VPS/服务器"
Write-Host "6. 编译所有平台"

$choice = Read-Host "请输入选项 (1-6)"

# 2. 定义平台配置函数
function Get-PlatformConfig($choice) {
    switch ($choice) {
        "1" { return @{os="windows"; arch="amd64"; out="flowcollect_windows_amd64.exe"; dir="windows"; desc="Windows (AMD64)"} }
        "2" { return @{os="linux"; arch="arm64"; out="flowcollect_android_arm64"; dir="android"; desc="Android (ARM64)"} }
        "3" { return @{os="linux"; arch="amd64"; out="flowcollect_openwrt_x86_64"; dir="openwrt-x86"; desc="OpenWrt x86_64"} }
        "4" { return @{os="linux"; arch="mipsle"; out="flowcollect_openwrt_mipsle"; dir="openwrt-mipsle"; desc="OpenWrt MIPSLE"} }
        "5" { return @{os="linux"; arch="amd64"; out="flowcollect_linux_amd64"; dir="linux"; desc="Linux (AMD64)"} }
    }
    return $null
}

# 3. 编译函数
function Build-ForPlatform($config) {
    Write-Host "`n正在编译: $($config.desc)..." -ForegroundColor Green

    # 设置环境变量
    $env:GOOS = $config.os
    $env:GOARCH = $config.arch

    # 特殊处理MIPS平台
    if ($config.desc -eq "OpenWrt MIPSLE") {
        $env:GOMIPS = "softfloat"
    }

    # 创建输出目录
    $outputDir = "$baseDir\$($config.dir)"
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

    # 输出文件路径
    $outputPath = "$outputDir\$($config.out)"

    # 编译时注入的变量
    $ldflags = "-s -w " +
            "-X 'main.DeviceID=$targetID-$($config.dir)' " +
            "-X 'main.MihomoAPIAddr=$apiAddr' " +
            "-X 'main.MihomoSecret=$secret' " +
            "-X 'main.BuildTime=$timestamp' " +
            "-X 'main.BuildVersion=$version'"

    # 执行编译
    Write-Host "目标文件: $outputPath"
    Write-Host "编译参数: GOOS=$($config.os) GOARCH=$($config.arch)"

    # 构建命令参数
    $buildArgs = @()
    $buildArgs += "build"
    $buildArgs += "-tags"
    $buildArgs += "client"
    $buildArgs += "-ldflags"
    $buildArgs += $ldflags
    $buildArgs += "-o"
    $buildArgs += $outputPath
    $buildArgs += "./client"

    # 执行编译
    & go @buildArgs

    # 检查编译结果
    if ($LASTEXITCODE -eq 0 -and (Test-Path $outputPath)) {
        # 计算文件大小和哈希
        $fileInfo = Get-Item $outputPath
        $fileSize = [math]::Round($fileInfo.Length / 1KB, 2)
        $fileHash = (Get-FileHash -Path $outputPath -Algorithm SHA256).Hash.Substring(0, 12)

        Write-Host "✓ 编译成功!" -ForegroundColor Green
        Write-Host "  文件: $($config.out)" -ForegroundColor Gray
        Write-Host "  大小: ${fileSize} KB" -ForegroundColor Gray
        Write-Host "  哈希: $fileHash" -ForegroundColor Gray
        Write-Host "  路径: $outputPath" -ForegroundColor Gray

        return @{
            Path = $outputPath
            Name = $config.out
            Size = $fileSize
            Hash = $fileHash
            Desc = $config.desc
        }
    } else {
        Write-Host "✗ 编译失败" -ForegroundColor Red
        return $null
    }
}

# 4. GitHub Release 发布函数
function Publish-ToGitHub($version, $baseDir) {
    Write-Host "`n=== GitHub Release 发布 ===" -ForegroundColor Cyan

    # 检查是否安装 GitHub CLI
    try {
        $ghVersion = gh --version
        Write-Host "✓ GitHub CLI 已安装" -ForegroundColor Green
    } catch {
        Write-Host "✗ GitHub CLI 未安装" -ForegroundColor Red
        Write-Host "请先安装 GitHub CLI: https://cli.github.com/" -ForegroundColor Yellow
        return $false
    }

    # 检查是否已登录
    try {
        $ghStatus = gh auth status 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "✗ GitHub CLI 未登录" -ForegroundColor Red
            Write-Host "请先执行: gh auth login" -ForegroundColor Yellow
            return $false
        }
        Write-Host "✓ GitHub CLI 已登录" -ForegroundColor Green
    } catch {
        Write-Host "✗ 检查登录状态失败" -ForegroundColor Red
        return $false
    }

    # 检查是否为 Git 仓库
    if (-not (Test-Path ".git")) {
        Write-Host "✗ 当前目录不是 Git 仓库" -ForegroundColor Red
        return $false
    }

    # 获取仓库信息
    try {
        $remoteUrl = git remote get-url origin
        Write-Host "✓ Git 仓库: $remoteUrl" -ForegroundColor Green
    } catch {
        Write-Host "✗ 无法获取 Git 远程仓库信息" -ForegroundColor Red
        return $false
    }

    # 询问发布标题和说明
    Write-Host "`n请输入 Release 信息:" -ForegroundColor Yellow
    $releaseTitle = Read-Host "标题 [默认: Release $version]"
    if ($releaseTitle -eq "") { $releaseTitle = "Release $version" }

    $releaseNotes = Read-Host "说明 [默认: 自动发布的 FlowCollect 版本]"
    if ($releaseNotes -eq "") { $releaseNotes = "自动发布的 FlowCollect 版本" }

    # 收集所有要上传的文件
    $filesToUpload = @()
    Get-ChildItem -Path $baseDir -Recurse -File | ForEach-Object {
        $filesToUpload += $_.FullName
    }

    if ($filesToUpload.Count -eq 0) {
        Write-Host "✗ 没有找到可上传的文件" -ForegroundColor Red
        return $false
    }

    Write-Host "`n准备上传以下文件:" -ForegroundColor Yellow
    foreach ($file in $filesToUpload) {
        $relativePath = $file.Replace("$baseDir\", "")
        Write-Host "  - $relativePath" -ForegroundColor Gray
    }

    # 确认发布
    $confirm = Read-Host "`n确认发布到 GitHub Release? (y/n) [默认: n]"
    if ($confirm -ne "y" -and $confirm -ne "Y") {
        Write-Host "已取消发布" -ForegroundColor Yellow
        return $false
    }

    # 创建 Release
    Write-Host "`n正在创建 GitHub Release..." -ForegroundColor Green

    try {
        # 检查是否已存在该标签
        $existingRelease = gh release view $tagName 2>$null

        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Release $tagName 已存在，更新中..." -ForegroundColor Green
            $ghArgs = @("release", "upload", $tagName, "--clobber")
        } else {
            Write-Host "✓ 创建新的 Release: $tagName" -ForegroundColor Green
            $ghArgs = @("release", "create", $tagName, "--title", $releaseTitle, "--notes", $releaseNotes)
        }

        # 添加文件
        foreach ($file in $filesToUpload) {
            $ghArgs += $file
        }

        # 执行发布
        Write-Host "执行命令: gh $ghArgs" -ForegroundColor Gray
        & gh @ghArgs

        if ($LASTEXITCODE -eq 0) {
            # 获取 Release URL
            $releaseUrl = gh release view $tagName --json url --jq '.url' 2>$null
            if ($releaseUrl) {
                Write-Host "`n✓ 发布成功!" -ForegroundColor Green
                Write-Host "Release URL: $releaseUrl" -ForegroundColor Cyan
            } else {
                Write-Host "`n✓ 发布成功!" -ForegroundColor Green
            }
            return $true
        } else {
            Write-Host "✗ 发布失败" -ForegroundColor Red
            return $false
        }

    } catch {
        Write-Host "✗ 发布过程中出错: $_" -ForegroundColor Red
        return $false
    }
}

# 5. 执行编译
$compiledFiles = @()

if ($choice -eq "6") {
    # 编译所有平台
    Write-Host "`n开始编译所有平台..." -ForegroundColor Cyan

    foreach ($i in 1..5) {
        $config = Get-PlatformConfig $i.ToString()
        if ($config) {
            $result = Build-ForPlatform $config
            if ($result) {
                $compiledFiles += $result
            }
        }
    }
} else {
    # 编译单个平台
    $config = Get-PlatformConfig $choice
    if ($config) {
        $result = Build-ForPlatform $config
        if ($result) {
            $compiledFiles += $result
        }
    } else {
        Write-Host "无效选项" -ForegroundColor Red
        exit 1
    }
}

# 6. 清理环境变量
$env:GOOS = $null
$env:GOARCH = $null
$env:GOMIPS = $null

# 7. 生成汇总信息
if ($compiledFiles.Count -gt 0) {
    Write-Host "`n" + ("="*60) -ForegroundColor Cyan
    Write-Host "编译完成汇总" -ForegroundColor Cyan
    Write-Host ("="*60) -ForegroundColor Cyan

    Write-Host "版本号: $version" -ForegroundColor Green
    Write-Host "编译时间: $timestamp" -ForegroundColor Green
    Write-Host "输出目录: $baseDir" -ForegroundColor Green
    Write-Host "设备ID: $targetID" -ForegroundColor Gray
    Write-Host "API地址: $apiAddr" -ForegroundColor Gray
    Write-Host "`n已编译文件:" -ForegroundColor Yellow

    foreach ($file in $compiledFiles) {
        Write-Host "  [$($file.Desc)]" -ForegroundColor White
        Write-Host "    → $($file.Name)" -ForegroundColor Gray
        Write-Host "    → 大小: $($file.Size) KB, 哈希: $($file.Hash)" -ForegroundColor Gray
    }

    # 创建版本信息文件（使用 UTF-8 编码）
    $infoFile = "$baseDir\build-info.txt"
    $infoContent = @"
FlowCollect 编译信息
====================
版本号: $version
编译时间: $timestamp
设备ID: $targetID
Mihomo API: $apiAddr

已编译文件:
"@

    foreach ($file in $compiledFiles) {
        $infoContent += "`n- $($file.Desc)"
        $infoContent += "  文件: $($file.Name)"
        $infoContent += "  大小: $($file.Size) KB"
        $infoContent += "  SHA256: $($file.Hash)..."
    }

    # 使用 UTF-8 编码写入文件
    [System.IO.File]::WriteAllText($infoFile, $infoContent, [System.Text.Encoding]::UTF8)
    Write-Host "`n版本信息已保存: $infoFile" -ForegroundColor Green

    # 创建校验和文件（使用 UTF-8 编码）
    $checksumFile = "$baseDir\checksums.txt"
    $checksumContent = ""
    foreach ($file in $compiledFiles) {
        $fullHash = (Get-FileHash -Path $file.Path -Algorithm SHA256).Hash
        $checksumContent += "$fullHash  $($file.Name)`n"
    }

    [System.IO.File]::WriteAllText($checksumFile, $checksumContent, [System.Text.Encoding]::UTF8)
    Write-Host "校验和文件: $checksumFile" -ForegroundColor Green

    # 生成快速部署脚本示例（使用 UTF-8 编码）
    $deployScript = "$baseDir\deploy-example.sh"
    $scriptContent = @"
#!/bin/bash
# FlowCollect 快速部署脚本
# 版本: $version
# 生成时间: $timestamp

echo 'FlowCollect 部署指南'
echo '===================='
echo '1. 将对应平台的可执行文件上传到目标设备'
echo '2. 添加执行权限 (Linux/Android/OpenWrt):'
echo '   chmod +x flowcollect_*'
echo '3. 运行程序:'
echo '   ./flowcollect_*'
echo ''
echo '运行时参数已编译到程序中，无需额外配置。'
echo ''
echo '验证版本信息:'
echo '   ./flowcollect_* --version'
echo ''
echo '查看帮助:'
echo '   ./flowcollect_* --help'
"@

    [System.IO.File]::WriteAllText($deployScript, $scriptContent, [System.Text.Encoding]::UTF8)
    Write-Host "部署指南: $deployScript" -ForegroundColor Green

    # 8. 询问是否发布到 GitHub Release
    Write-Host "`n" + ("="*60) -ForegroundColor Cyan
    Write-Host "GitHub Release 发布选项" -ForegroundColor Cyan
    Write-Host ("="*60) -ForegroundColor Cyan

    $publishChoice = Read-Host "是否发布到 GitHub Release? (y/n) [默认: n]"
    if ($publishChoice -eq "y" -or $publishChoice -eq "Y") {
        $publishResult = Publish-ToGitHub -version $version -baseDir $baseDir
        if ($publishResult) {
            Write-Host "`n✓ 所有操作完成!" -ForegroundColor Green
        }
    } else {
        Write-Host "已跳过 GitHub Release 发布" -ForegroundColor Yellow
    }

    Write-Host "`n下一步操作:" -ForegroundColor Yellow
    Write-Host "1. 将对应平台的文件上传到目标设备" -ForegroundColor Gray
    Write-Host "2. Linux/OpenWrt/Android 需要: chmod +x 可执行文件" -ForegroundColor Gray
    Write-Host "3. 直接运行即可，参数已编译到程序中" -ForegroundColor Gray
    Write-Host "`n完成!" -ForegroundColor Green
} else {
    Write-Host "`n没有成功编译任何文件。" -ForegroundColor Red
}