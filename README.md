# Go HTTP & FTP File Server

两个用 Go 编写的轻量文件服务器，各自独立、均可编译为单个可执行文件，用于在局域网内快速把一台主机上的目录共享出去。

| 组件 | 协议 | 默认端口 | 适用场景 |
| --- | --- | --- | --- |
| [FtpFileServer](FtpFileServer/README.md) | FTP | `2121` | 用 FileZilla、Windows 资源管理器等现成 FTP 客户端访问；GIS 软件可通过 `/vsicurl/ftp://` 按需读取 |
| [HttpFileServer](HttpFileServer/README.md) | HTTP / REST | `8080` | 浏览器直接浏览和下载；给程序对接用的 REST API；顺带把目录里的静态网页托管出去 |

两个组件互不依赖，可以只部署其中一个，也可以分别升级。它们沿用同一套命令行风格、JSON 配置格式，以及只读模式与日志开关的约定。

## 功能对照

| 能力 | FtpFileServer | HttpFileServer |
| --- | :---: | :---: |
| 账号密码认证 | ✅ | ✅（HTTP Basic Authentication） |
| 匿名访问 | ✅ | ✅ |
| 只读模式（禁止写入） | ✅ `--readonly` | ✅ `--readonly` |
| JSON 配置文件 | ✅ `--config` | ✅ `--config` |
| 诊断日志 | ✅ `--info` | ✅ `--info` |
| 上传 / 下载文件 | ✅ | ✅ |
| 新建目录、重命名、删除 | ✅ | ✅ |
| 断点续传 / 随机读取 | ✅ `REST STREAM`、`SIZE`、`MDTM` | ✅ 标准 HTTP `Range` |
| 机器可读的目录列表 | ✅ `MLSD`、`MLST` | ✅ `GET /files` 返回 JSON |
| 图形界面 | 无，使用任意 FTP 客户端 | 内置文件浏览网页 |
| 静态网页托管 | 不适用 | ✅ 按扩展名返回真实 MIME，可直接当网页打开 |
| 页内文本预览 | 不适用 | ✅ 前 100 KB，自动识别 UTF-8 / GB18030 |
| 自定义 URL 前缀 | 不适用 | ✅ `--url-prefix` |
| 第三方依赖 | 内置 `third_party/server`（goftp/server，MIT） | 无，仅 Go 标准库 |
| 交叉编译 | 免 CGO | 免 CGO |

两个程序都支持 `--version` 查询当前版本，各自的 README 中记录了版本号与对应的功能变更约定。

## 快速开始

需要 Go 1.20 或更高版本。先把待共享的目录准备好，`--root` 指向的目录必须已存在。

### FtpFileServer

```bash
cd FtpFileServer
go build -o ftp-file-server.exe .
./ftp-file-server.exe --root ./ftp-data --username admin --password secret
```

服务监听 `2121` 控制端口，并使用 `30000-30100` 作为被动模式数据端口。用 FTP 客户端连接 `ftp://127.0.0.1:2121` 即可。**若在另一台机器上连接，必须同时在防火墙放行控制端口和整段被动端口**，否则能登录但无法列目录和传输。

### HttpFileServer

```bash
cd HttpFileServer
go build -o http-file-server.exe ./cmd/httpfileserver
./http-file-server.exe --root ./http-data --username admin --password secret
```

浏览器打开 `http://127.0.0.1:8080/` 即可浏览文件；REST API 的基础路径为 `http://127.0.0.1:8080/api/v1`，接口定义见 [HttpFileServer/API.md](HttpFileServer/API.md)。

网页的文件列表按类别显示图标，文件名直接指向文件的真实路径，因此保存下来的就是真实文件名，右键也能复制下载地址；网页、图片、文本、PDF 等浏览器可以直接显示的类型点击即在浏览器打开，其它类型点击下载。点击「预览」可在当前页面查看文件的前 100 KB 文本内容。

### 不配置账号密码时

两者都会切换到匿名访问，启动时输出风险警告。匿名且非只读时任何能连上端口的客户端都能上传、重命名和删除文件。仅应在可信内网或 VPN 中使用。

## 目录结构

```text
.
├── FtpFileServer/                  FTP 文件服务器
│   ├── main.go                     程序入口、启动信息与信号处理
│   ├── config.go                   命令行参数与 JSON 配置文件解析
│   ├── server.go                   FTP 服务装配（认证、被动端口、日志）
│   ├── driver.go                   文件系统驱动：路径越界与符号链接校验、只读拦截、随机读取
│   ├── version.go                  版本号
│   ├── README.md                   使用、编译、部署与防火墙说明
│   └── third_party/server/         内置 FTP 协议库（goftp/server，MIT 许可证）
└── HttpFileServer/                 HTTP 文件服务器
    ├── cmd/httpfileserver/main.go  程序入口与命令行解析
    ├── internal/fileservice/
    │   ├── server.go               路由、认证、路径校验、REST API 实现
    │   └── web/index.html          内嵌的文件浏览网页（HTML/CSS/JS 全部内联）
    ├── README.md                   使用、编译、部署与防火墙说明
    └── API.md                      REST API 定义
```

HttpFileServer 的网页资源通过 Go `embed` 编入可执行文件，部署时不需要额外复制静态资源目录。

## 编译与部署

本机编译、Linux 交叉编译（Windows / Linux amd64、Linux arm64、Linux armv7）、systemd 服务单元、以及 Windows Defender / firewalld / UFW / nftables 的防火墙规则示例，都在各组件 README 中给出：

- [FtpFileServer 编译与部署](FtpFileServer/README.md#windows-本机编译)
- [HttpFileServer 编译与部署](HttpFileServer/README.md#windows-本机编译)

## 安全须知

- **两个协议都是明文协议。** FTP 的用户名、密码和文件内容在网络上明文传输；HTTP 的 Basic Authentication 在纯 HTTP 上同样等同明文凭据。它们都只适合部署在可信内网或 VPN 中；需要跨公网访问时，应由反向代理或网关终结 TLS，并把服务端口限制在受控网段内可达。
- **配置文件中的密码为明文。** 请限制配置文件权限，不要提交到源代码仓库（本仓库的 `.gitignore` 已忽略 `*-file-server.json`）。
- FTP 需要控制端口与被动数据端口同时可达；只放行控制端口会导致目录列表和文件传输失败。
- 两个服务都以启动进程的权限读写 `--root` 指定的目录，建议使用专用低权限账户运行。

## 许可证

本项目采用 [MIT 许可证](LICENSE)。

`FtpFileServer/third_party/server` 目录内置了 [goftp/server](https://gitea.com/goftp/server) 的源码（FTP 协议实现），按其 MIT 许可证分发，许可证原文见 [FtpFileServer/third_party/server/LICENSE](FtpFileServer/third_party/server/LICENSE)。
