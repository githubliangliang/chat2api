# Windows 原生部署与管理员登录故障排查

本文记录 Windows amd64 单二进制部署中的管理员登录、时区启动和认证接口问题，以及当前版本的修复行为。

## 1. 故障现象

常见现象包括：

- 在 `config.yaml` 中填写了 `admin_email` 和 `admin_password`，但无法登录。
- 浏览器访问 `/login` 后跳转到 `/setup`。
- `POST /api/v1/auth/login` 返回 `404 Not Found`。
- 启动日志显示：

```text
Failed to initialize application: invalid timezone "Asia/Shanghai": unknown time zone Asia/Shanghai
```

这些现象可能同时出现，但原因不同，需要按启动状态分别判断。

## 2. 先确认运行目录

Windows 原生模式下，程序使用启动时的工作目录和数据目录。不要只修改 `deploy\config.yaml` 后从其他目录启动，并期待程序自动读取该文件。

推荐使用一个独立的数据目录，例如：

```text
sub2api_1.1.5_windows_amd64\
  sub2api.exe
  config.yaml
  data\
    sub2api.db
```

在 PowerShell 中进入二进制所在目录后启动：

```powershell
cd E:\Google_download\sub2api\sub2api_1.1.5_windows_amd64
.\sub2api.exe
```

检查当前实际使用的配置和数据库：

```powershell
Get-Location
Get-Item .\config.yaml
Get-Item .\data\sub2api.db -ErrorAction SilentlyContinue
```

## 3. 配置管理员初始化

当前版本在数据库为空时，会使用配置中的以下字段创建一次管理员：

```yaml
default:
  admin_email: "admin@example.com"
  admin_password: "CHANGE_ME_STRONG_PASSWORD"
```

只有邮箱和密码都填写时才会创建管理员；已有任意用户时不会覆盖或重置账号密码。密码必须为 8 到 128 个字符，并且必须把 `CHANGE_ME_STRONG_PASSWORD` 示例值改成真实强密码。配置文件中的管理员字段只在首次引导时生效，后续修改不会改变已有账号。

首次初始化方式：

1. 启动程序。
2. 如果配置文件已包含 `default.admin_email` 和 `default.admin_password`，程序会在空数据库中自动创建管理员。
3. 如果未填写这两个字段，打开 `http://127.0.0.1:8080/setup`，完成向导。
4. 使用创建的管理员邮箱和密码访问 `http://127.0.0.1:8080/login`。

也可以使用 CLI 向导（适合不使用配置文件自动初始化时）：

```powershell
Stop-Process -Name sub2api -Force -ErrorAction SilentlyContinue
.\sub2api.exe -setup
```

## 4. Windows 时区启动错误

### 4.1 原因

`Asia/Shanghai` 是 IANA 时区名称。旧版 Windows 发布包未嵌入 Go IANA 时区数据库时，Go 的 `time.LoadLocation("Asia/Shanghai")` 会失败。

Windows 系统本身使用的时区标识是 `China Standard Time`，不是 `Asia/Shanghai`。

### 4.2 修复

当前版本已经将 Go IANA 时区数据库嵌入单文件，优先直接保留配置：

```yaml
timezone: Asia/Shanghai
```

只有旧版本仍报错时，才临时设置 `TIMEZONE=Local`，让程序使用 Windows 当前本地时区：

```powershell
[Environment]::SetEnvironmentVariable('TIMEZONE', 'Local', 'User')
```

设置后必须新开一个 PowerShell 窗口，再启动程序：

```powershell
cd E:\Google_download\sub2api\sub2api_1.1.5_windows_amd64
.\sub2api.exe
```

也可以仅对当前 PowerShell 会话设置：

```powershell
$env:TIMEZONE = 'Local'
.\sub2api.exe
```

注意：`TIMEZONE` 主要用于旧版本或自动安装兼容路径；正常启动会读取配置文件中的 `timezone`。

检查环境变量：

```powershell
[Environment]::GetEnvironmentVariable('TIMEZONE', 'User')
```

预期输出：

```text
Local
```

## 5. `POST /api/v1/auth/login` 返回 404

### 情况 A：仍处于 Setup 模式

先检查：

```powershell
$r = Invoke-WebRequest -UseBasicParsing `
  'http://127.0.0.1:8080/setup/status' `
  -SkipHttpErrorCheck
$r.Content
```

如果返回：

```json
{"data":{"needs_setup":true}}
```

说明管理员初始化尚未完成。此时 `/login` 会重定向到 `/setup`，认证 API 尚未注册，访问 `/api/v1/auth/login` 返回 404 是预期行为。请先完成 Setup 向导。

### 情况 B：Setup 已完成但仍返回 404

当前版本会在 Setup 完成后自动关闭 Setup 服务并启动主服务，不需要手动重启。若仍返回 404，优先检查启动日志和端口占用；如果运行的是旧版本，再手动重启：

```powershell
Stop-Process -Name sub2api -Force -ErrorAction SilentlyContinue
$env:TIMEZONE = 'Local'
Start-Process -FilePath '.\sub2api.exe' `
  -WorkingDirectory (Get-Location).Path
```

## 6. 完整恢复流程

以下流程适用于本地 Windows 单二进制部署，默认保留现有数据库：

```powershell
cd E:\Google_download\sub2api\sub2api_1.1.5_windows_amd64

# 1. 停止旧进程
Stop-Process -Name sub2api -Force -ErrorAction SilentlyContinue

# 2. 启动程序
$p = Start-Process -FilePath '.\sub2api.exe' `
  -WorkingDirectory (Get-Location).Path `
  -WindowStyle Hidden `
  -PassThru
$p.Id
```

如果是首次安装，打开：

```text
http://127.0.0.1:8080/setup
```

当前版本完成向导后会自动切换到主服务，不需要再次重启。只有旧版本才需要手动执行上述重启命令。

## 7. 验证清单

### 7.1 进程是否运行

```powershell
Get-Process sub2api -ErrorAction SilentlyContinue |
  Select-Object Id, StartTime, Responding, Path
```

`Responding` 应为 `True`。

### 7.2 Setup 是否完成

```powershell
$r = Invoke-WebRequest -UseBasicParsing `
  'http://127.0.0.1:8080/setup/status' `
  -SkipHttpErrorCheck
$r.Content
```

完成后应包含：

```json
"needs_setup":false
```

### 7.3 登录路由是否已注册

使用错误凭据进行无害探测：

```powershell
$body = '{"email":"nobody@example.invalid","password":"wrong-password"}'
$r = Invoke-WebRequest -UseBasicParsing `
  'http://127.0.0.1:8080/api/v1/auth/login' `
  -Method Post `
  -ContentType 'application/json' `
  -Body $body `
  -SkipHttpErrorCheck
"Status: $($r.StatusCode)"
$r.Content
```

判断方式：

| 返回结果 | 含义 |
|---|---|
| `404 Not Found` | Setup 模式仍在运行，或应用启动失败 |
| `401 invalid email or password` | 登录路由正常，账号或密码不正确 |
| `200` | 登录成功 |

## 8. 常见误区

### 修改了错误的配置文件

`deploy\config.yaml` 是部署模板；程序实际使用哪一份配置取决于启动目录、数据目录和环境变量。排查时应先确认 `Get-Location`、`config.yaml` 和 `data\sub2api.db` 的实际路径。

### 只改 `TZ`，不改 `TIMEZONE`

当前版本已将 Go IANA 时区数据库嵌入 Windows 单文件，`Asia/Shanghai` 等时区无需额外安装。旧版本遇到时区错误时，可临时设置 `TIMEZONE=Local`；`TZ` 只在自动安装环境中作为兼容变量读取。

### 运行的是旧二进制

如果日志仍显示 Setup 完成后没有进入主服务，确认启动的是最新构建；当前代码会自动切换路由，不要求手动重启。

### 删除数据库解决问题

不建议为了登录问题直接删除 `data\sub2api.db`。数据库中包含管理员、账号、API Key 和系统设置。优先修复时区、启动目录和安装状态；只有明确需要全新安装时才备份后重建数据库。

## 9. 最终判断标准

部署正常需要同时满足：

1. `sub2api.exe` 进程持续运行。
2. 配置时区可以直接使用 `Asia/Shanghai`；旧版本兼容场景才需要 `TIMEZONE=Local`。
3. `/setup/status` 返回 `needs_setup: false`。
4. `POST /api/v1/auth/login` 不再返回 404。
5. 使用 Setup 向导或配置文件创建的管理员邮箱和密码可以登录。
