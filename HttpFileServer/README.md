# HTTP File Server

一个使用 Go 标准库编写的 HTTP 文件服务器。它提供文件 REST API 和内嵌的文件浏览网页，支持账号认证或匿名访问、文件浏览、上传、下载、建目录、移动/重命名和删除。

服务直接使用 HTTP Basic Authentication，不提供 HTTPS 终结能力。仅应部署在可信内网或 VPN 中；生产环境应由反向代理提供 HTTPS。

详细 REST 协议见 [API.md](API.md)。

## 命令行参数

```text
http-file-server --root <文件根目录> [--config <文件>] [--username <用户名> --password <密码>] [--port <端口>] [--url-prefix <路径>] [--readonly] [--info] [-h|--help] [--version]
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `--port` | 否 | HTTP 监听端口，范围为 `1` 到 `65535`，默认 `8080`。 |
| `--username` | 否 | HTTP Basic Authentication 用户名。必须与 `--password` 同时提供；两者均不提供时启用匿名访问。 |
| `--password` | 否 | HTTP Basic Authentication 密码。必须与 `--username` 同时提供；两者均不提供时启用匿名访问。 |
| `--root` | 是 | 对客户端暴露的本地根目录。网页和 API 中的根目录均对应此目录；目录必须已存在。 |
| `--config` | 否 | JSON 配置文件路径。命令行中显式提供的参数覆盖配置文件中的同名配置。 |
| `--url-prefix` | 否 | 页面、REST API 和直接访问的 URL 前缀，默认空。指定 `/storage` 后，网页入口为 `/storage/`，API 为 `/storage/api/v1/...`，直接访问为 `/storage/<文件相对路径>`。 |
| `--readonly` | 否 | 仅允许浏览、列目录和下载；上传、移动、删除和建目录请求返回 `403`，网页也不显示写操作。 |
| `--info` | 否 | 输出 HTTP 请求、来源地址和响应状态等诊断日志；未指定时仅输出错误。 |
| `-h`、`--help` | 否 | 显示命令行帮助信息并退出。 |
| `--version` | 否 | 输出程序版本并退出；当前版本为 `2.2.0`，无需提供 `--root`。 |

Windows 示例：

```powershell
.\http-file-server.exe --username admin --password secret --root D:\http-data
```

匿名只读示例：

```powershell
.\http-file-server.exe --root D:\http-data --readonly
```

匿名模式不校验客户端提供的凭据，任何能访问服务端口的客户端都可访问文件。匿名且非只读时，任何访问者都可通过网页或 API 上传、移动、删除和建目录；程序启动时会给出高风险警告。仅应在可信内网或 VPN 中使用匿名访问。

### 版本

当前版本为 `2.2.0`。执行 `--version` 可直接查询版本，无需配置数据根目录：

```powershell
.\http-file-server.exe --version
```

功能行为发生修改时必须同步递增版本号，并更新本 README 与 [API.md](API.md)。

### URL 前缀与网页

默认网页入口是 `http://<主机>:8080/`，默认 API 基础路径是 `http://<主机>:8080/api/v1`。设置前缀时，两者同时移动：

```powershell
.\http-file-server.exe --root D:\http-data --url-prefix /storage
```

此时网页入口为 `http://<主机>:8080/storage/`，API 基础路径为 `http://<主机>:8080/storage/api/v1`。访问不带末尾 `/` 的 `/storage` 会重定向到 `/storage/`。

网页资源通过 Go `embed` 编入可执行文件，不依赖外部 HTML、CSS 或 JavaScript 文件。已配置用户名和密码时，浏览器会显示原生 HTTP Basic Authentication 登录框；网页和 API 使用相同的认证和权限规则。

网页的文件列表为每个条目显示类别图标（目录、影像、矢量、图纸、三维模型、点云、表格、演示、文档、代码、数据、压缩包、音视频、可执行程序等，按扩展名判定，未识别的一律使用通用文件图标）。文件名本身是指向文件相对路径的链接，末段即文件名，可直接分享；可内联类型（网页、样式、脚本、文本、图片、PDF 等，与“直接文件访问”的判定一致）点击后在浏览器打开，因此放在数据目录里的静态网页可以当网页看；其它类型点击即下载，保存名就是真实文件名；两者都能右键复制下载地址。目录名仍为按钮，单击进入目录。点击“预览”在当前页面弹出文本预览，最多读取并显示文件的前 100 KB，超出时在弹窗中提示文件总大小和截断位置；可内联类型的弹窗中额外提供“打开”。预览按 UTF-8 解码，失败时回退 GB18030；文件包含二进制内容或编码无法识别时会在弹窗中明确提示，而不是静默显示乱码。

除网页根路径和 REST API 外，服务还支持通过文件相对路径直接访问普通文件。例如数据根目录中存在 `模型/2_mesh/metadata.osgb` 时，可使用 `http://<主机>:8080/模型/2_mesh/metadata.osgb` 访问。设置 `--url-prefix /storage` 时，地址为 `http://<主机>:8080/storage/模型/2_mesh/metadata.osgb`。该方式仅支持 `GET` 与 `HEAD`，且仅返回普通文件；目录、符号链接、非法路径或不存在的文件均返回 `404`，不会提供目录浏览。

该方式不设置 `Content-Disposition`（URL 末段已是文件名），但会按扩展名返回 `Content-Type` 并带 `X-Content-Type-Options: nosniff`：网页（`html`、`htm`、`xhtml`）、样式与脚本（`css`、`js`、`mjs`）、数据与文本（`json`、`xml`、`txt`、`md`、`log`、`csv`）、图片（`svg`、`png`、`jpg`、`jpeg`、`gif`、`webp`、`bmp`、`ico`、`avif`）、`pdf`、字体（`woff`、`woff2`、`ttf`）浏览器直接打开，因此放在数据目录里的静态网页连同同级样式、脚本、图片都能直接正常访问；其它扩展名一律返回 `application/octet-stream`，由浏览器下载。API 的 `GET /api/v1/file` 恒为 `application/octet-stream` 加 `Content-Disposition: attachment`，需要强制下载时使用 API 地址。

`--url-prefix` 必须以 `/` 开头，不能包含空路径段、`.`、`..`、反斜杠、查询串或片段。`/` 与空值等价，都会规范化为无前缀。

### JSON 配置文件

使用 `--config` 指定 JSON 配置文件。除 `--config` 本身外，所有运行参数均可写入配置文件；字段均为可选，未填写时使用程序默认值。示例 `http-file-server.json`：

```json
{
  "port": 8080,
  "username": "admin",
  "password": "secret",
  "root": "./http-data",
  "readonly": false,
  "info": false,
  "url_prefix": "/storage"
}
```

启动：

```powershell
.\http-file-server.exe --config D:\http-server\http-file-server.json
```

配置文件中的相对 `root` 按配置文件所在目录解析；命令行中的相对 `--root` 按当前工作目录解析。命令行覆盖配置文件中的对应字段，包括显式设为 `false` 的布尔值：

```powershell
.\http-file-server.exe --config D:\http-server\http-file-server.json --port 8081 --readonly=true --info=false
```

配置文件中的密码为明文。应限制该文件的访问权限，不要提交到源代码仓库。配置文件未同时给出 `username` 和 `password` 时会启用匿名访问；只提供其中一个会导致启动失败。

### 启动信息和日志

每次启动都会输出监听端口、文件存储根目录和 URL 前缀，例如：

```text
HTTP server configuration:
  Listening port: 8080
  Storage root: D:\http-data
  URL prefix: /storage
```

默认日志保持安静，仅输出错误。增加 `--info` 后，才会输出每个请求的方法、URI、来源地址和响应状态：

```powershell
.\http-file-server.exe --username viewer --password secret --root D:\http-data --readonly --info
```

## Windows 本机编译

要求 Windows 已安装 Go 1.20 或更高版本。在项目根目录执行：

```powershell
go test ./...
$env:CGO_ENABLED = "0"
go build -trimpath -buildvcs=false -ldflags="-s -w" -o http-file-server.exe ./cmd/httpfileserver
Remove-Item Env:CGO_ENABLED
```

`CGO_ENABLED=0` 生成不依赖 C 运行时的 Go 程序；`-trimpath` 删除本机源码路径；`-ldflags="-s -w"` 移除符号表和调试信息。它们共同减小发布文件并避免嵌入 Git 元数据。

## Linux 交叉编译

以下命令在 Linux Shell 的项目根目录执行：

```bash
# 设置国内代理
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn

mkdir -p dist
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/http-file-server-windows-amd64.exe ./cmd/httpfileserver
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/http-file-server-linux-amd64 ./cmd/httpfileserver
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/http-file-server-linux-arm64 ./cmd/httpfileserver
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/http-file-server-linux-armv7 ./cmd/httpfileserver
```

所有交叉编译命令都禁用 CGO，因此无需安装目标平台的 C 交叉编译器。

## Linux 部署

以下示例将程序部署到 `/opt/http-file-server`，数据目录设为 `/srv/http-data`：

```bash
sudo install -d -m 755 /opt/http-file-server
sudo install -d -m 750 /srv/http-data
sudo install -m 755 http-file-server-linux-amd64 /opt/http-file-server/http-file-server
```

以当前用户后台运行：

```bash
cd /opt/http-file-server
nohup ./http-file-server --port 8080 --username admin --password 'secret' --root /srv/http-data >/dev/null 2>&1 &
echo $! > http-file-server.pid
```

停止后台进程：

```bash
kill "$(cat /opt/http-file-server/http-file-server.pid)"
```

命令行参数可能被同机有权限的用户查看。生产环境应限制主机访问权限，并优先使用仅本机可读的 systemd 服务文件或受保护的配置文件。

### 使用 systemd 部署

创建专用账户和目录：

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin httpfileserver
sudo install -d -o httpfileserver -g httpfileserver -m 750 /srv/http-data
sudo install -d -m 755 /opt/http-file-server
sudo install -m 755 http-file-server-linux-amd64 /opt/http-file-server/http-file-server
```

创建 `/etc/systemd/system/http-file-server.service`，并替换账号、密码和目录：

```ini
[Unit]
Description=HTTP File Server
After=network.target

[Service]
Type=simple
User=httpfileserver
Group=httpfileserver
WorkingDirectory=/opt/http-file-server
ExecStart=/opt/http-file-server/http-file-server --port 8080 --username admin --password secret --root /srv/http-data
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/srv/http-data

[Install]
WantedBy=multi-user.target
```

限制服务文件权限并启用服务：

```bash
sudo chown root:root /etc/systemd/system/http-file-server.service
sudo chmod 600 /etc/systemd/system/http-file-server.service
sudo systemctl daemon-reload
sudo systemctl enable --now http-file-server
sudo systemctl status http-file-server
```

## 防火墙配置

HTTP 服务只使用一个 TCP 监听端口。部署前确认服务已监听：

```bash
sudo ss -lntp | grep ':8080'
```

Windows Defender 防火墙示例，仅允许可信网段 `172.16.41.0/24` 访问：

```powershell
New-NetFirewallRule -DisplayName "HTTP File Server" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8080 -RemoteAddress 172.16.41.0/24
Get-NetFirewallRule -DisplayName "HTTP File Server" | Get-NetFirewallPortFilter | Format-Table Protocol, LocalPort, RemotePort
Remove-NetFirewallRule -DisplayName "HTTP File Server"
```

firewalld 示例：

```bash
sudo firewall-cmd --state  # 或者使用 firewall-cmd 查看
sudo firewall-cmd --list-all  # 查看所有已开放的端口和服务

sudo firewall-cmd --zone=public --add-port=8080/tcp --permanent
sudo firewall-cmd --reload
sudo firewall-cmd --zone=public --query-port=8080/tcp
```

UFW 示例：

```bash
sudo ufw allow from 192.168.1.0/24 to any port 8080 proto tcp
sudo ufw status numbered
```

请将端口和网段替换为实际部署值。不要在公网直接暴露明文 HTTP Basic Authentication；应由 HTTPS 反向代理终结 TLS，并将服务端端口限制在反向代理或受控网络可达范围内。
