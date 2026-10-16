# FTP File Server

一个 Go 编写的 FTP 文件服务器，支持账号认证或匿名访问，并兼容 FileZilla 等常见 FTP 客户端的文件浏览、上传、下载、目录管理、重命名和删除操作。

服务使用明文 FTP，适用于可信内网或 VPN 网络；不支持 FTPS/TLS，也不适合直接暴露到公网。

## 命令行参数

```text
ftp-file-server --root <文件根目录> [--config <配置文件>] [--username <用户名> --password <密码>] [--port <端口>] [--readonly] [--info] [--pasv-min-port <端口> --pasv-max-port <端口>] [-h|--help]
ftp-file-server --version
```

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `--port` | 否 | FTP 控制连接端口，范围为 `1` 到 `65535`，默认 `2121`。 |
| `--username` | 否 | 登录用户名。必须与 `--password` 同时提供；二者都不提供时启用匿名访问。 |
| `--password` | 否 | 登录密码。必须与 `--username` 同时提供；二者都不提供时启用匿名访问。 |
| `--root` | 是 | 对客户端暴露的本地根目录。客户端看到的 `/` 即此目录。目录必须已存在。 |
| `--config` | 否 | JSON 配置文件路径。命令行中已提供的参数会覆盖配置文件中的同名配置。 |
| `--readonly` | 否 | 指定后仅允许浏览、列目录和下载；上传、删除、建目录和重命名等修改操作会被拒绝。 |
| `--info` | 否 | 输出 FTP 会话、请求和调试日志；未指定时仅输出错误。 |
| `--pasv-min-port` | 否 | 被动模式数据端口下界，默认 `30000`。 |
| `--pasv-max-port` | 否 | 被动模式数据端口上界，默认 `30100`。必须不小于最小端口。 |
| `--version` | 否 | 输出当前程序版本号并退出。 |
| `-h`、`--help` | 否 | 显示命令行帮助信息并退出。 |

### 版本号

当前版本为 `2.0.4`。使用以下命令查看：

```powershell
.\ftp-file-server.exe --version
```

后续每增加或修改一项功能，递增补丁版本号：`2.0.4`、`2.0.5`、`2.0.6`，依此类推。

Windows 示例：

```powershell
.\ftp-file-server.exe --username admin --password secret --root D:\ftp-data
```

匿名访问示例：

```powershell
.\ftp-file-server.exe --root D:\ftp-data
```

匿名模式不校验客户端提供的用户名和密码，任何能连接到服务端口的客户端均可访问文件。启动时会输出警告信息；除非同时指定 `--readonly`，匿名客户端也拥有写入、删除和重命名权限。仅应在可信内网或 VPN 中使用匿名访问。

默认会使用 `30000-30100` 作为被动模式数据端口范围。若需更改范围，必须同时在服务器防火墙中放行相同范围：

```powershell
.\ftp-file-server.exe --port 2121 --username admin --password secret --root D:\ftp-data --pasv-min-port 30000 --pasv-max-port 30100
```

### JSON 配置文件

使用 `--config` 指定 JSON 配置文件。除 `--config` 本身外，其他所有运行参数均可写入配置文件；字段均为可选，未填写时使用程序默认值。示例 `ftp-file-server.json`：

```json
{
  "port": 2121,
  "username": "admin",
  "password": "secret",
  "root": "./ftp-data",
  "readonly": false,
  "info": false,
  "pasv_min_port": 30000,
  "pasv_max_port": 30100
}
```

启动：

```powershell
.\ftp-file-server.exe --config D:\ftp-server\ftp-file-server.json
```

配置文件中的 `root` 为相对路径时，按配置文件所在目录解析。命令行中的 `--root` 为相对路径时，仍按当前工作目录解析。

命令行覆盖配置文件中对应的字段，包括显式设置布尔值为 `false`：

```powershell
.\ftp-file-server.exe --config D:\ftp-server\ftp-file-server.json --port 2221 --readonly=true --info=false
```

配置文件中的密码为明文。应限制该文件访问权限，且不要将其提交到源代码仓库。若配置文件未包含 `username` 和 `password`，将启用匿名访问。

### 启动信息和日志

每次启动都会输出监听端口、文件存储根目录和被动模式端口范围，例如：

```text
FTP server configuration:
  Listening port: 2121
  Storage root: D:\ftp-data
  Passive ports: 30000-30100
```

默认保持会话日志静默，仅输出错误。增加 `--info` 后，才输出客户端连接、登录、FTP 命令和响应等诊断信息：

```powershell
.\ftp-file-server.exe --username admin --password secret --root D:\ftp-data --info
```

只读示例：

```powershell
.\ftp-file-server.exe --port 2121 --username viewer --password secret --root D:\ftp-data --readonly
```

## Windows 本机编译

要求 Windows 已安装 Go 1.20 或更高版本。以下命令在 `cmd.exe` 中、项目根目录下执行：

### 配置国内模块镜像

以下设置仅对当前 CMD 窗口有效，适合在下载依赖或编译前执行：

```cmd
set "GOPROXY=https://goproxy.cn,direct"
set "GOSUMDB=sum.golang.google.cn"
```

如需将该设置保存到 Go 的全局配置中，执行：

```cmd
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
```

编译命令会自动使用以上代理设置：

```cmd
go mod download
set "CGO_ENABLED=0"
go build -trimpath -buildvcs=false -ldflags="-s -w" -o ftp-file-server.exe .
set "CGO_ENABLED="
```

`CGO_ENABLED=0` 生成不依赖 C 运行时的静态 Go 程序；`-trimpath` 删除本机源码路径；`-ldflags="-s -w"` 移除符号表和调试信息。它们共同减小发布文件并避免嵌入 Git 元数据。

## Linux 交叉编译

以下命令均在 Linux Shell 中执行。先创建输出目录：

```bash
mkdir -p dist
```

Windows x86_64：

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/ftp-file-server-windows-amd64.exe .
```

Linux x86_64：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/ftp-file-server-linux-amd64 .
```

Linux ARM64：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/ftp-file-server-linux-arm64 .
```

Linux ARMv7（32 位）：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -buildvcs=false -ldflags="-s -w" -o dist/ftp-file-server-linux-armv7 .
```

所有交叉编译命令都禁用 CGO，因此无需安装目标平台的 C 交叉编译器。

若已安装 [UPX](https://upx.github.io/)，可进一步压缩发布文件：

```bash
upx --best --lzma dist/ftp-file-server-linux-amd64
```

UPX 不是运行必需条件。使用前应在目标发行版上验证启动和 FTP 客户端连接。

## Linux 直接部署

以下示例将程序和数据分别部署到 `/opt/ftp-file-server` 与 `/srv/ftp-data`：

```bash
sudo install -d -m 755 /opt/ftp-file-server
sudo install -d -m 750 /srv/ftp-data
sudo install -m 755 ftp-file-server-linux-amd64 /opt/ftp-file-server/ftp-file-server
```

以当前用户后台运行且不输出日志：

```bash
cd /opt/ftp-file-server
nohup ./ftp-file-server --port 2121 --username admin --password 'secret' --root /srv/ftp-data >/dev/null 2>&1 &
echo $! > ftp-file-server.pid
```

停止该后台进程：

```bash
kill "$(cat /opt/ftp-file-server/ftp-file-server.pid)"
```

命令行参数可被同机有权限的用户查看。生产环境应限制主机访问权限，并使用仅本机可读的 systemd 服务文件。

## 使用 systemd 部署

创建专用的系统账号和目录：

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin ftpfileserver
sudo install -d -o ftpfileserver -g ftpfileserver -m 750 /srv/ftp-data
sudo install -d -m 755 /opt/ftp-file-server
sudo install -m 755 ftp-file-server-linux-amd64 /opt/ftp-file-server/ftp-file-server
```

创建 `/etc/systemd/system/ftp-file-server.service`，将用户名、密码、端口和根目录替换为实际值：

```ini
[Unit]
Description=FTP File Server
After=network.target

[Service]
Type=simple
User=ftpfileserver
Group=ftpfileserver
WorkingDirectory=/opt/ftp-file-server
ExecStart=/opt/ftp-file-server/ftp-file-server --port 2121 --username admin --password secret --root /srv/ftp-data
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/srv/ftp-data
StandardOutput=null
StandardError=null

[Install]
WantedBy=multi-user.target
```

## FTP 协议能力与 GDAL/QGIS 兼容性

项目通过 `go.mod` 的 `replace` 指令使用 `third_party/server` 中兼容 Go 1.20
的旧版 FTP 库源码。服务端已实现并在 `FEAT` 中声明 `REST STREAM`、`SIZE`、
`MDTM`、`MLSD` 和 `MLST`。

`MLSD` 通过数据连接返回机器可读的目录条目，`MLST` 通过控制连接返回单个
文件或目录的元数据。`REST <offset>` 配合 `RETR <file>` 使用文件驱动的
Seek 能力，支持 GDAL/QGIS 的随机读取。

启动服务后可检查能力列表：

```bash
curl -v --user 'admin:secret' ftp://127.0.0.1:2121/ -o /dev/null
```

验证随机读取（每条命令应下载 64 字节）：

```bash
curl -v --user 'admin:secret' --range 0-63 \
  'ftp://127.0.0.1:2121/DEM/NK45E001009DEMK.img' -o /dev/null
curl -v --user 'admin:secret' --range 4096-4159 \
  'ftp://127.0.0.1:2121/DEM/NK45E001009DEMK.img' -o /dev/null
```

当前协议兼容性版本为 `2.0.4`。

文件下载（`RETR`）使用跨平台的缓冲写入路径，不依赖旧版库的文件到网络
直传优化路径。服务端记录下载失败时的原始错误、文件路径、偏移量和已传输
字节数，便于排查文件读取错误与客户端断开数据连接两类问题。

### GDAL/QGIS 按需读取

GDAL/QGIS 的 `/vsicurl/ftp://` 可能只读取文件前部数据后主动关闭数据连接，
并发送 `ABOR` 结束传输。该场景与操作系统无关：服务端会先返回
`426 Connection closed; transfer aborted`，随后对 `ABOR` 返回
`226 Abort successful`，不会将客户端中止误报为 `551 Error reading file`。

当 `REST` 使用非零偏移时，`RETR` 的传输长度为文件的剩余字节数，超出文件
范围的偏移会被拒绝。该协议流程已通过 QGIS/GDAL 的实际访问验证。

限制服务文件权限并启用服务：

```bash
sudo chown root:root /etc/systemd/system/ftp-file-server.service
sudo chmod 600 /etc/systemd/system/ftp-file-server.service
sudo systemctl daemon-reload
sudo systemctl enable --now ftp-file-server
sudo systemctl status ftp-file-server
```

常用管理命令：

```bash
sudo systemctl restart ftp-file-server
sudo systemctl stop ftp-file-server
sudo systemctl disable --now ftp-file-server
```

服务文件将标准输出和错误输出都丢弃，不会写入日志。若排查启动失败，可临时删除 `StandardOutput=null` 和 `StandardError=null`，执行 `sudo systemctl daemon-reload && sudo systemctl restart ftp-file-server` 后用 `journalctl -u ftp-file-server` 查看错误。

## 防火墙配置

先确认服务已监听控制端口（以下以 `2121` 为例）：

```bash
sudo ss -lntp | grep ':2121'
```

服务默认使用控制端口 `2121/tcp`，以及被动模式数据端口 `30000-30100/tcp`。如通过参数修改被动端口范围，以下规则中的端口范围也必须同步修改。

### Windows Server / Windows Defender 防火墙

以管理员 PowerShell 查看防火墙配置文件状态：

```powershell
Get-NetFirewallProfile | Format-Table Name, Enabled, DefaultInboundAction
```

仅允许可信客户端网段 `172.16.41.0/24` 访问控制端口和被动数据端口。请替换为实际客户端网段：

```powershell
New-NetFirewallRule -DisplayName "FTP File Server Control" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 2121 -RemoteAddress 172.16.41.0/24
New-NetFirewallRule -DisplayName "FTP File Server Passive Data" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 30000-30100 -RemoteAddress 172.16.41.0/24
Get-NetFirewallRule -DisplayName "FTP File Server*" | Get-NetFirewallPortFilter | Format-Table Protocol, LocalPort, RemotePort
```

删除规则：

```powershell
Remove-NetFirewallRule -DisplayName "FTP File Server Control"
Remove-NetFirewallRule -DisplayName "FTP File Server Passive Data"
```

### CentOS、RHEL、Rocky Linux（firewalld）

查看防火墙状态、活动区域和已有规则：

```bash
sudo firewall-cmd --state  # 或者使用 firewall-cmd 查看
sudo firewall-cmd --list-all  # 查看所有已开放的端口和服务
sudo firewall-cmd --get-active-zones
sudo firewall-cmd --zone=public --list-all
```

将 `public` 替换为网卡实际所在区域后，允许 FTP 控制端口并永久保存：

```bash
sudo firewall-cmd --zone=public --add-port=2121/tcp --permanent
sudo firewall-cmd --zone=public --add-port=30000-30100/tcp --permanent
sudo firewall-cmd --reload
sudo firewall-cmd --zone=public --query-port=2121/tcp
sudo firewall-cmd --zone=public --query-port=30000-30100/tcp
```

删除该规则：

```bash
sudo firewall-cmd --zone=public --remove-port=2121/tcp --permanent
sudo firewall-cmd --zone=public --remove-port=30000-30100/tcp --permanent
sudo firewall-cmd --reload
```

### Ubuntu、Debian（UFW）

查看 UFW 状态和现有规则：

```bash
sudo ufw status verbose
sudo ufw status numbered
```

仅允许受信任内网 `192.168.1.0/24` 访问控制端口。请替换为实际客户端网段：

```bash
sudo ufw allow from 192.168.1.0/24 to any port 2121 proto tcp
sudo ufw allow from 192.168.1.0/24 to any port 30000:30100 proto tcp
sudo ufw status numbered
```

删除时使用 `sudo ufw status numbered` 获取规则编号，再执行：

```bash
sudo ufw delete <规则编号>
```

未启用 UFW 的远程主机不要直接执行 `sudo ufw enable`，应先确认 SSH 管理端口已被允许，以免断开远程管理连接。

### 使用 nftables 的系统

查看当前规则集：

```bash
sudo nft list ruleset
```

若系统已有 `inet filter input` 链，可用以下规则仅允许受信任网段访问控制端口；网段应替换为实际值：

```bash
sudo nft add rule inet filter input ip saddr 192.168.1.0/24 tcp dport 2121 accept
sudo nft add rule inet filter input ip saddr 192.168.1.0/24 tcp dport 30000-30100 accept
sudo nft list chain inet filter input
```

此命令添加的规则在重启后是否保留取决于发行版的 nftables 配置方式，应按本机的 `/etc/nftables.conf` 或配置管理流程持久化，避免直接覆盖已有规则集。

### 被动模式注意事项

FTP 除控制端口外，还需要被动模式数据连接。默认数据端口范围为 `30000-30100/tcp`；仅放行 `2121/tcp` 后，登录可能成功，但目录列表、上传或下载会失败。防火墙规则必须同时放行控制端口和被动数据端口范围。

建议仅在可信内网或 VPN 内部署，并将防火墙访问源限制为客户端网段。若部署在 NAT 或公网环境，还需要让 PASV 响应返回客户端可达的公网地址；当前版本尚未提供该地址配置。

## 网络说明

FTP 文件传输使用被动数据连接，默认端口范围为 `30000-30100`。在主机防火墙中应放行控制端口和该完整范围；在 NAT 或公网环境中，当前版本不支持配置 PASV 响应中的公网地址，请优先部署在内网或 VPN 中。
