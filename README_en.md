# Go HTTP & FTP File Server

[中文](README.md) | **English**

Two lightweight file servers written in Go. They are independent of each other and each compiles to a single executable, so you can share a directory on one machine across a local network in seconds.

| Component | Protocol | Default port | When to use it |
| --- | --- | --- | --- |
| [FtpFileServer](FtpFileServer/README.md) | FTP | `2121` | Access with an existing FTP client such as FileZilla or Windows File Explorer; GIS software can read files on demand through `/vsicurl/ftp://` |
| [HttpFileServer](HttpFileServer/README.md) | HTTP / REST | `8080` | Browse and download in a browser; a REST API for program integration; also serves static web pages from the directory |

The two components have no dependency on each other. You can deploy only one of them, or upgrade them separately. They share the same command-line style, the same JSON configuration format, and the same conventions for read-only mode and diagnostic logging.

## Feature Comparison

| Capability | FtpFileServer | HttpFileServer |
| --- | :---: | :---: |
| Username / password authentication | ✅ | ✅ (HTTP Basic Authentication) |
| Anonymous access | ✅ | ✅ |
| Read-only mode (writes disabled) | ✅ `--readonly` | ✅ `--readonly` |
| JSON configuration file | ✅ `--config` | ✅ `--config` |
| Diagnostic logging | ✅ `--info` | ✅ `--info` |
| Upload / download files | ✅ | ✅ |
| Create directory, rename, delete | ✅ | ✅ |
| Resume / random access | ✅ `REST STREAM`, `SIZE`, `MDTM` | ✅ Standard HTTP `Range` |
| Machine-readable directory listing | ✅ `MLSD`, `MLST` | ✅ `GET /files` returns JSON |
| Graphical interface | None; use any FTP client | Built-in file browser page |
| Static web page hosting | Not applicable | ✅ Real MIME type by extension, pages open directly in the browser |
| In-page text preview | Not applicable | ✅ First 100 KB, UTF-8 / GB18030 auto-detected |
| Custom URL prefix | Not applicable | ✅ `--url-prefix` |
| Third-party dependencies | Vendored `third_party/server` (goftp/server, MIT) | None, Go standard library only |
| Cross-compilation | CGO-free | CGO-free |

Both programs support `--version`. Each component README records its own version number and the rule for bumping it when behavior changes.

## Quick Start

Go 1.20 or later is required. Prepare the directory you want to share first; the directory passed to `--root` must already exist.

### FtpFileServer

```bash
cd FtpFileServer
go build -o ftp-file-server.exe .
./ftp-file-server.exe --root ./ftp-data --username admin --password secret
```

The server listens on control port `2121` and uses `30000-30100` as the passive-mode data port range. Connect an FTP client to `ftp://127.0.0.1:2121`. **When connecting from another machine, you must open both the control port and the whole passive port range in the firewall**, otherwise the client can log in but cannot list directories or transfer files.

### HttpFileServer

```bash
cd HttpFileServer
go build -o http-file-server.exe ./cmd/httpfileserver
./http-file-server.exe --root ./http-data --username admin --password secret
```

Open `http://127.0.0.1:8080/` in a browser to browse files. The REST API is rooted at `http://127.0.0.1:8080/api/v1`; see [HttpFileServer/API.md](HttpFileServer/API.md) for the full definition.

The file list shows a category icon per entry, and each file name links directly to the file's real path, so the saved file keeps its real name and the download URL can be copied from the context menu. Types the browser can display on its own — web pages, images, text, PDF — open directly in the browser when clicked; other types are downloaded. Clicking "Preview" shows the first 100 KB of the file as text in the current page.

### When no credentials are configured

Both servers fall back to anonymous access and print a risk warning at startup. With anonymous access and a writable root, any client that can reach the port can upload, rename, and delete files. Use this only on a trusted local network or inside a VPN.

## Repository Layout

```text
.
├── FtpFileServer/                  FTP file server
│   ├── main.go                     Entry point, startup output, signal handling
│   ├── config.go                   Command-line flags and JSON configuration parsing
│   ├── server.go                   FTP service assembly (auth, passive ports, logging)
│   ├── driver.go                   Filesystem driver: path escape and symlink checks, read-only guard, random access
│   ├── version.go                  Version number
│   ├── README.md                   Usage, build, deployment, and firewall notes (Chinese)
│   └── third_party/server/         Vendored FTP protocol library (goftp/server, MIT license)
└── HttpFileServer/                 HTTP file server
    ├── cmd/httpfileserver/main.go  Entry point and command-line parsing
    ├── internal/fileservice/
    │   ├── server.go               Routing, authentication, path validation, REST API
    │   └── web/index.html          Embedded file browser page (HTML/CSS/JS all inline)
    ├── README.md                   Usage, build, deployment, and firewall notes (Chinese)
    └── API.md                      REST API definition (Chinese)
```

The HttpFileServer web assets are compiled into the executable with Go `embed`, so no static asset directory has to be copied at deployment time.

The component READMEs are written in Chinese; this file is the English entry point and mirrors them section by section.

## Build and Deployment

Building locally, cross-compiling to Linux (Windows / Linux amd64, Linux arm64, Linux armv7), systemd unit files, and firewall rule examples for Windows Defender / firewalld / UFW / nftables are all covered in the component READMEs:

- [FtpFileServer build and deployment](FtpFileServer/README.md#windows-本机编译)
- [HttpFileServer build and deployment](HttpFileServer/README.md#windows-本机编译)

## Security Notes

- **Both protocols are plaintext.** FTP sends user names, passwords, and file contents over the network in the clear; HTTP Basic Authentication over plain HTTP is equally plaintext credentials. Deploy these servers only on a trusted local network or inside a VPN. For access across the public internet, terminate TLS at a reverse proxy or gateway and keep the service ports reachable only from controlled network segments.
- **Passwords in configuration files are plaintext.** Restrict file permissions and do not commit them to a source repository (this repository's `.gitignore` already ignores `*-file-server.json`).
- FTP requires both the control port and the passive data port range to be reachable; opening only the control port makes directory listings and transfers fail.
- Both servers read and write the `--root` directory with the privileges of the process that starts them. Running them under a dedicated low-privilege account is recommended.

## License

This project is released under the [MIT License](LICENSE).

The `FtpFileServer/third_party/server` directory contains the source of [goftp/server](https://gitea.com/goftp/server) (the FTP protocol implementation), distributed under its MIT license. The license text is at [FtpFileServer/third_party/server/LICENSE](FtpFileServer/third_party/server/LICENSE).
