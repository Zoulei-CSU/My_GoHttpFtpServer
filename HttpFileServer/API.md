# HTTP File Server API

本文定义 HTTP File Server 的 REST API。运行与部署说明见 [README.md](README.md)。

## 基础约定

默认 API 基础路径为：

```text
http://<主机>:<端口>/api/v1
```

指定 `--url-prefix /storage` 后，基础路径变为：

```text
http://<主机>:<端口>/storage/api/v1
```

文中所有路径均以该 API 基础路径为前缀。请求和响应使用 UTF-8；错误响应的内容类型为 `application/json; charset=utf-8`，格式为：

```json
{"error":"错误说明"}
```

### 认证

同时配置 `--username` 和 `--password` 时，网页和全部 API 均要求 HTTP Basic Authentication。未认证或凭据错误时服务返回 `401`、错误 JSON 和 `WWW-Authenticate: Basic realm="FileStorageManager"` 响应头。

未配置账号密码时，服务为匿名模式，不校验 `Authorization` 请求头。匿名与 `--readonly` 的组合决定是否允许写入。

### `path` 查询参数

除移动接口外，所有文件和目录接口都使用 URL 编码的 `path` 查询参数。其值必须是服务根目录内、以 `/` 分隔的相对路径，例如 `DOM/XAFW/a.shp`。移动接口中的 `source` 和 `destination` 使用同一规则。

禁止绝对路径、反斜杠、`.`、`..`、空路径段和请求路径中已有的符号链接。违反规则返回 `400`。`GET /files` 的空 `path` 表示服务根目录；其他接口的空 `path` 返回 `400`。

## 接口

### 列出目录

```http
GET /files?path=<目录相对路径>
```

`path` 可为空以列出根目录。成功时返回 `200`：

```json
{
  "entries": [
    {
      "name": "DOM",
      "directory": true,
      "size": 0,
      "modified": "2026-08-05T03:12:00Z",
      "etag": ""
    },
    {
      "name": "a.txt",
      "directory": false,
      "size": 7,
      "modified": "2026-08-05T03:12:00Z",
      "etag": "\"17e8c48f0\""
    }
  ]
}
```

`entries` 始终为数组。`modified` 为 UTC 时间；目录的 `etag` 为空字符串，文件 ETag 由文件大小与修改时间生成。

### 下载文件与获取属性

```http
GET  /file?path=<文件相对路径>
HEAD /file?path=<文件或目录相对路径>
```

`GET` 仅接受文件，成功时返回 `200`、`Content-Type: application/octet-stream`、`ETag` 和 `Content-Disposition: attachment` 响应头；实现使用标准 HTTP Range 支持，因此可处理断点续传请求。请求目录返回 `400`。

`Content-Disposition` 的取值为 RFC 6266 格式：文件名为纯 ASCII 时直接给出 `filename=<文件名>`，包含非 ASCII 字符时按 RFC 5987 编码为 `filename*=utf-8''<百分号编码文件名>`。因此客户端按文件名保存时得到服务端上的真实文件名；本接口的 URL 末段固定为 `file`，不携带文件名。该响应头对客户端是附加信息，不改变状态码、`Content-Length`、`ETag` 与 Range 语义，`HEAD` 请求同样返回。

`HEAD` 请求文件时返回与下载相同的文件元数据但无响应体。请求目录时返回 `204` 和 `X-File-Directory: true`，用于客户端区分目录与文件。

### 直接文件访问

除 API 外，还可使用服务根路径下的文件相对路径直接访问普通文件：

```http
GET  /模型/2_mesh/metadata.osgb
HEAD /模型/2_mesh/metadata.osgb
```

此方式会将 URL 路径映射到服务根目录中的文件；设置 `--url-prefix /storage` 时，对应地址为 `/storage/模型/2_mesh/metadata.osgb`。网页根路径 `/`（或 `/storage/`）仍返回嵌入网页，`/api/v1/...` 仍为 API。

该方式的 URL 末段已是文件名，浏览器可自行取名，因此不设置 `Content-Disposition`，与 API 下载接口不同；`Content-Type` 按扩展名给出，并带 `X-Content-Type-Options: nosniff`：

- **可内联类型**（`html`、`htm`、`xhtml`、`css`、`js`、`mjs`、`json`、`xml`、`txt`、`md`、`log`、`csv`、`svg`、`png`、`jpg`、`jpeg`、`gif`、`webp`、`bmp`、`ico`、`avif`、`pdf`、`woff`、`woff2`、`ttf`）返回对应类型，浏览器直接打开。因此放在根目录下的静态网页及其同级样式、脚本、图片可直接通过浏览器访问。
- **其它扩展名**一律返回 `application/octet-stream`，浏览器按下载处理，文件名取 URL 末段。

与 API 的 `GET /file` 相比：API 恒为 `application/octet-stream` 加 `Content-Disposition: attachment`（强制下载），直接访问则按扩展名决定呈现方式。需要强制下载时使用 API 地址。

直接下载仅支持普通文件以及 `GET`、`HEAD` 方法，支持 HTTP Range。目录、符号链接、非法路径和不存在的文件均返回 `404`；其他方法返回 `405`。已配置用户名和密码时，未经 Basic Authentication 验证的请求返回 `401`。

### 上传文件

```http
PUT /file?path=<目标文件相对路径>&overwrite=true
Content-Type: application/octet-stream

<原始文件字节>
```

父目录必须已经存在。目标不存在时成功返回 `201`。目标已存在时，只有 `overwrite=true` 才允许覆盖；未设置或值不是 `true` 时返回 `409`。目标为目录时同样返回 `409`。

上传内容先写入目标目录内的临时文件，完成后通过重命名提交，以避免中断上传留下半成品目标文件。

### 新建目录

```http
PUT /directory?path=<目录相对路径>
```

服务会递归创建缺失的父目录。成功时返回 `204`，目录已经存在时也返回 `204`。

### 移动或重命名文件、目录

```http
POST /file/move
Content-Type: application/json

{
  "source": "DOM/incoming/a.txt",
  "destination": "DOM/archive/2026/a.txt",
  "overwrite": false,
  "create_parents": true
}
```

服务在本地文件系统内使用重命名操作直接移动，不经由客户端读取或重新上传文件内容。`source` 与 `destination` 均为必填的服务根目录内相对路径；二者不能相同，目录也不能移动到自身内部。

`create_parents=true` 时递归创建目标父目录；未设置或为 `false` 时，目标父目录不存在返回 `404`。目标已存在时，只有 `overwrite=true` 才允许替换，否则返回 `409`。成功返回 `204`。只读模式下返回 `403`。

### 删除文件或目录

```http
DELETE /file?path=<文件或空目录相对路径>
DELETE /file?path=<目录相对路径>&recursive=true
```

删除文件或空目录成功返回 `204`。删除非空目录必须设置 `recursive=true`；该参数会递归删除该目录内的全部内容，操作不可恢复。目标不存在返回 `404`。

## 状态码

| 状态码 | 含义 |
| --- | --- |
| `200` | 成功读取目录或下载文件。 |
| `201` | 文件上传成功。 |
| `204` | 目录属性检查、建目录、移动或删除成功。 |
| `400` | 方法对应的参数无效、路径无效、移动到自身内部，或将下载接口用于目录。 |
| `401` | 未提供或提供了错误的 Basic Authentication 凭据。 |
| `403` | 服务处于只读模式，或操作系统拒绝访问。 |
| `404` | 请求的路径、移动源路径，或上传/移动目标父目录不存在。 |
| `405` | 不支持的 HTTP 方法。 |
| `409` | 上传或移动目标已存在但未允许覆盖，或上传目标为目录。 |
| `500` | 其他本地文件系统操作失败，例如删除非空目录但未使用递归删除。 |

## 网页服务

网页入口为服务根路径：未设置 URL 前缀时为 `/`，设置 `/storage` 时为 `/storage/`。网页使用本 API 完成浏览、下载和可写模式下的文件操作；它不是另一套协议，也不会绕过认证、路径校验或只读限制。

网页的文件列表按扩展名为每个条目显示类别图标，并为文件提供“预览”按钮。文件名是指向文件相对路径的链接，末段即文件名，可以直接分享；可内联类型（见“直接文件访问”）点击后在浏览器打开，其它类型点击即下载，两者都可右键复制下载地址；目录名为按钮，单击进入目录。预览通过 `GET /file` 携带 `Range: bytes=0-102399` 读取前 100 KB，因此在服务端的表现与普通下载请求一致，仅访问一个正常文件、不产生额外接口；服务端不需要、也不提供预览专用接口。文件大于 100 KB 时网页只显示已读取的部分并提示文件总大小。
