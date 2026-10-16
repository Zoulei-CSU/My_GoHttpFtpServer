package fileservice

import (
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Server struct {
	root      string
	username  string
	password  string
	readOnly  bool
	urlPrefix string
}

type listEntry struct {
	Name      string    `json:"name"`
	Directory bool      `json:"directory"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	ETag      string    `json:"etag"`
}

type listResponse struct {
	Entries []listEntry `json:"entries"`
}

//go:embed web/index.html
var fileBrowserHTML []byte

func NewServer(root, username, password string, readOnly bool, urlPrefix string) (*Server, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("root must not be empty")
	}
	if (username == "") != (password == "") {
		return nil, errors.New("username and password must be provided together")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absRoot, 0750); err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, err
	}
	if err := validateURLPrefix(urlPrefix); err != nil {
		return nil, err
	}
	return &Server{root: filepath.Clean(realRoot), username: username, password: password, readOnly: readOnly, urlPrefix: urlPrefix}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	if s.urlPrefix != "" {
		mux.HandleFunc(s.urlPrefix, s.redirectBrowserRoot)
	}
	mux.HandleFunc(s.browserPath(), s.browserOrDirectDownload)
	mux.HandleFunc(s.apiPath("files"), s.list)
	mux.HandleFunc(s.apiPath("file"), s.file)
	mux.HandleFunc(s.apiPath("file/move"), s.move)
	mux.HandleFunc(s.apiPath("directory"), s.directory)
	return s.authenticate(mux)
}

func (s *Server) browserPath() string {
	return s.urlPrefix + "/"
}

func (s *Server) redirectBrowserRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.urlPrefix {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, s.browserPath(), http.StatusPermanentRedirect)
}

func (s *Server) browser(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.browserPath() {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	page := strings.ReplaceAll(string(fileBrowserHTML), "{{FILE_BROWSER_CONFIG}}", s.browserConfigJSON())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, page)
}

// browserConfig 是注入网页的运行时配置，字段名与网页中的读取方式一致。
type browserConfig struct {
	ReadOnly         bool     `json:"readOnly"`
	InlineExtensions []string `json:"inlineExtensions"`
}

func (s *Server) browserConfigJSON() string {
	payload, err := json.Marshal(browserConfig{ReadOnly: s.readOnly, InlineExtensions: inlineExtensions()})
	if err != nil {
		return `{"readOnly":true,"inlineExtensions":[]}`
	}
	return string(payload)
}

func (s *Server) browserOrDirectDownload(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == s.browserPath() {
		s.browser(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path, err := s.resolvePath(strings.TrimPrefix(r.URL.Path, s.browserPath()), false)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	// 直接下载的 URL 末段就是文件名，浏览器可自行取名，行为保持不变。
	s.download(w, r, path, false)
}

func (s *Server) apiPath(endpoint string) string {
	return s.urlPrefix + "/api/v1/" + endpoint
}

func validateURLPrefix(value string) error {
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.ContainsAny(value, "\\?#") {
		return errors.New("invalid URL prefix")
	}
	for _, segment := range strings.Split(strings.TrimPrefix(value, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("invalid URL prefix")
		}
	}
	return nil
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.username == "" {
			next.ServeHTTP(w, r)
			return
		}
		username, password, ok := r.BasicAuth()
		validUser := subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) == 1
		validPassword := subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) == 1
		if !ok || !validUser || !validPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="FileStorageManager"`)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) resolvePath(value string, allowRoot bool) (string, error) {
	if value == "" {
		if allowRoot {
			return s.root, nil
		}
		return "", errors.New("path must not be empty")
	}
	if strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return "", errors.New("path must be a relative slash-separated path")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("path contains an invalid segment")
		}
	}
	parts := strings.Split(value, "/")
	target := filepath.Join(s.root, filepath.FromSlash(value))
	relative, err := filepath.Rel(s.root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes the service root")
	}
	current := s.root
	for _, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			break
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("symbolic links are not permitted")
		}
	}
	return target, nil
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	directory, err := s.resolvePath(r.URL.Query().Get("path"), true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		writeFileError(w, err)
		return
	}
	response := listResponse{Entries: make([]listEntry, 0, len(entries))}
	for _, item := range entries {
		info, err := item.Info()
		if err != nil {
			writeFileError(w, err)
			return
		}
		entry := listEntry{Name: item.Name(), Directory: item.IsDir(), Size: info.Size(), Modified: info.ModTime().UTC()}
		if !entry.Directory {
			entry.ETag = etag(info)
		}
		response.Entries = append(response.Entries, entry)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) file(w http.ResponseWriter, r *http.Request) {
	path, err := s.resolvePath(r.URL.Query().Get("path"), false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// API 下载路径的 URL 末段固定为 file，必须由响应头给出真实文件名。
		s.download(w, r, path, true)
	case http.MethodPut:
		if s.rejectWrite(w) {
			return
		}
		s.upload(w, r, path)
	case http.MethodDelete:
		if s.rejectWrite(w) {
			return
		}
		s.remove(w, r, path)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type moveRequest struct {
	Source        string `json:"source"`
	Destination   string `json:"destination"`
	Overwrite     bool   `json:"overwrite"`
	CreateParents bool   `json:"create_parents"`
}

func (s *Server) move(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.rejectWrite(w) {
		return
	}

	var request moveRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if request.Source == "" || request.Destination == "" {
		writeError(w, http.StatusBadRequest, "source and destination are required")
		return
	}
	source, err := s.resolvePath(request.Source, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	destination, err := s.resolvePath(request.Destination, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if sourceInfo.IsDir() {
		relative, err := filepath.Rel(source, destination)
		if err != nil || (relative != ".." && relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
			writeError(w, http.StatusBadRequest, "destination cannot be inside source directory")
			return
		}
	}

	parent := filepath.Dir(destination)
	if request.CreateParents {
		if err := os.MkdirAll(parent, 0750); err != nil {
			writeFileError(w, err)
			return
		}
	} else if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "destination parent directory does not exist")
		} else if err == nil {
			writeError(w, http.StatusBadRequest, "destination parent is not a directory")
		} else {
			writeFileError(w, err)
		}
		return
	}

	destinationInfo, err := os.Lstat(destination)
	if err == nil {
		if os.SameFile(sourceInfo, destinationInfo) {
			writeError(w, http.StatusBadRequest, "source and destination must differ")
			return
		}
		if !request.Overwrite {
			writeError(w, http.StatusConflict, "destination already exists")
			return
		}
	} else if !os.IsNotExist(err) {
		writeFileError(w, err)
		return
	}
	var backup string
	if destinationInfo != nil {
		temporary, err := os.CreateTemp(parent, ".http-file-server-move-")
		if err != nil {
			writeFileError(w, err)
			return
		}
		backup = temporary.Name()
		if err := temporary.Close(); err != nil {
			_ = os.Remove(backup)
			writeFileError(w, err)
			return
		}
		if err := os.Remove(backup); err != nil {
			writeFileError(w, err)
			return
		}
		if err := os.Rename(destination, backup); err != nil {
			writeFileError(w, err)
			return
		}
	}
	if err := os.Rename(source, destination); err != nil {
		if backup != "" {
			_ = os.Rename(backup, destination)
		}
		writeFileError(w, err)
		return
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) directory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.rejectWrite(w) {
		return
	}
	path, err := s.resolvePath(r.URL.Query().Get("path"), false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.MkdirAll(path, 0750); err != nil {
		writeFileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) rejectWrite(w http.ResponseWriter) bool {
	if !s.readOnly {
		return false
	}
	writeError(w, http.StatusForbidden, "file server is read-only")
	return true
}

func (s *Server) download(w http.ResponseWriter, r *http.Request, path string, attachment bool) {
	file, err := os.Open(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeFileError(w, err)
		return
	}
	if info.IsDir() {
		if r.Method == http.MethodHead {
			w.Header().Set("X-File-Directory", "true")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusBadRequest, "path is a directory")
		return
	}
	w.Header().Set("ETag", etag(info))
	if attachment {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", contentDisposition(info.Name()))
	} else {
		// 文件相对路径直接访问：按扩展名决定浏览器是内联呈现还是下载。
		w.Header().Set("Content-Type", contentType(info.Name()))
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

// inlineContentTypes 列出通过文件相对路径直接访问时，允许浏览器内联呈现的扩展名。
// 不在表中的扩展名一律按 application/octet-stream 返回，由浏览器下载，
// 避免未知格式被浏览器当作可渲染内容处理。
var inlineContentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".htm":   "text/html; charset=utf-8",
	".xhtml": "application/xhtml+xml; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".xml":   "application/xml; charset=utf-8",
	".txt":   "text/plain; charset=utf-8",
	".md":    "text/plain; charset=utf-8",
	".log":   "text/plain; charset=utf-8",
	".csv":   "text/plain; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".bmp":   "image/bmp",
	".ico":   "image/x-icon",
	".avif":  "image/avif",
	".pdf":   "application/pdf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
}

// contentType 返回文件相对路径直接访问时使用的 Content-Type；未列出的扩展名按二进制处理。
func contentType(name string) string {
	if value, exists := inlineContentTypes[strings.ToLower(filepath.Ext(name))]; exists {
		return value
	}
	return "application/octet-stream"
}

// inlineExtensions 返回可内联呈现的扩展名列表（不含点号，已排序），供网页判断点击行为。
func inlineExtensions() []string {
	extensions := make([]string, 0, len(inlineContentTypes))
	for extension := range inlineContentTypes {
		extensions = append(extensions, strings.TrimPrefix(extension, "."))
	}
	sort.Strings(extensions)
	return extensions
}

// contentDisposition 生成 RFC 6266 的 Content-Disposition 值，使客户端保存文件时
// 使用真实文件名。非 ASCII 名称按 RFC 5987 编码为 filename*（UTF-8 百分号编码），
// 现代浏览器据此还原原始名称；纯 ASCII 名称直接使用 filename 字段。
func contentDisposition(name string) string {
	value := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if value == "" {
		return "attachment"
	}
	if index := strings.Index(value, "filename*="); index >= 0 {
		return "attachment; " + value[index:]
	}
	return value
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request, path string) {
	if existing, err := os.Stat(path); err == nil {
		if existing.IsDir() || r.URL.Query().Get("overwrite") != "true" {
			writeError(w, http.StatusConflict, "destination already exists")
			return
		}
	} else if !os.IsNotExist(err) {
		writeFileError(w, err)
		return
	}

	parent := filepath.Dir(path)
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		writeError(w, http.StatusNotFound, "destination parent directory does not exist")
		return
	}
	temporary, err := os.CreateTemp(parent, ".upload-*")
	if err != nil {
		writeFileError(w, err)
		return
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := io.Copy(temporary, r.Body); err != nil {
		temporary.Close()
		writeFileError(w, err)
		return
	}
	if err := temporary.Close(); err != nil {
		writeFileError(w, err)
		return
	}
	if err := os.Rename(temporaryName, path); err != nil {
		writeFileError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) remove(w http.ResponseWriter, r *http.Request, path string) {
	info, err := os.Lstat(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if info.IsDir() && r.URL.Query().Get("recursive") == "true" {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		writeFileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func etag(info os.FileInfo) string {
	return fmt.Sprintf("\"%x\"", info.Size()^info.ModTime().UnixNano())
}

func writeFileError(w http.ResponseWriter, err error) {
	if os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "path does not exist")
		return
	}
	if os.IsPermission(err) {
		writeError(w, http.StatusForbidden, "permission denied")
		return
	}
	writeError(w, http.StatusInternalServerError, "file operation failed")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
