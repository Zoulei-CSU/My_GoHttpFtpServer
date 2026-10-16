package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"httpfileserver/internal/fileservice"
)

const version = "2.2.0"

var errVersion = errors.New("version requested")

type config struct {
	Port      int
	Root      string
	Username  string
	Password  string
	ReadOnly  bool
	Info      bool
	URLPrefix string
	Anonymous bool
}

type fileConfig struct {
	Port      *int    `json:"port"`
	Username  *string `json:"username"`
	Password  *string `json:"password"`
	Root      *string `json:"root"`
	ReadOnly  *bool   `json:"readonly"`
	Info      *bool   `json:"info"`
	URLPrefix *string `json:"url_prefix"`
}

func parseConfig(args []string) (config, error) {
	flags := flag.NewFlagSet("http-file-server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	result := config{Port: 8080}
	var configPath string
	var help bool
	var showVersion bool
	flags.IntVar(&result.Port, "port", 8080, "HTTP listening port")
	flags.StringVar(&result.Username, "username", "", "HTTP Basic Authentication username")
	flags.StringVar(&result.Password, "password", "", "HTTP Basic Authentication password")
	flags.StringVar(&result.Root, "root", "", "root directory exposed by HTTP")
	flags.BoolVar(&result.ReadOnly, "readonly", false, "disable all write operations")
	flags.BoolVar(&result.Info, "info", false, "enable HTTP request and diagnostic logs")
	flags.StringVar(&result.URLPrefix, "url-prefix", "", "URL path prefix for the web page, REST API, and direct downloads")
	flags.StringVar(&configPath, "config", "", "JSON configuration file")
	flags.BoolVar(&help, "help", false, "show this help message")
	flags.BoolVar(&help, "h", false, "show this help message")
	flags.BoolVar(&showVersion, "version", false, "show version information")
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if help {
		return config{}, flag.ErrHelp
	}
	if showVersion {
		return config{}, errVersion
	}
	overrides := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) { overrides[item.Name] = true })
	if configPath != "" {
		values, configDir, err := readConfigFile(configPath)
		if err != nil {
			return config{}, err
		}
		applyFileConfig(&result, values, configDir, overrides)
	}
	if result.Port < 1 || result.Port > 65535 {
		return config{}, errors.New("--port must be between 1 and 65535")
	}
	urlPrefix, err := normalizeURLPrefix(result.URLPrefix)
	if err != nil {
		return config{}, err
	}
	result.URLPrefix = urlPrefix
	if result.Username == "" && result.Password == "" {
		result.Anonymous = true
	} else if result.Username == "" || result.Password == "" {
		return config{}, errors.New("--username and --password must be provided together")
	}
	if result.Root == "" {
		return config{}, errors.New("--root is required")
	}
	root, err := filepath.Abs(result.Root)
	if err != nil {
		return config{}, fmt.Errorf("resolve --root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return config{}, fmt.Errorf("invalid --root: %w", err)
	}
	if !info.IsDir() {
		return config{}, errors.New("--root must name a directory")
	}
	result.Root = filepath.Clean(root)
	return result, nil
}

func readConfigFile(path string) (fileConfig, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fileConfig{}, "", fmt.Errorf("resolve --config: %w", err)
	}
	contents, err := os.ReadFile(absPath)
	if err != nil {
		return fileConfig{}, "", fmt.Errorf("read --config: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var result fileConfig
	if err := decoder.Decode(&result); err != nil {
		return fileConfig{}, "", fmt.Errorf("parse --config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fileConfig{}, "", errors.New("parse --config: multiple JSON values")
		}
		return fileConfig{}, "", fmt.Errorf("parse --config: %w", err)
	}
	return result, filepath.Dir(absPath), nil
}

func applyFileConfig(result *config, values fileConfig, configDir string, overrides map[string]bool) {
	if values.Port != nil && !overrides["port"] {
		result.Port = *values.Port
	}
	if values.Username != nil && !overrides["username"] {
		result.Username = *values.Username
	}
	if values.Password != nil && !overrides["password"] {
		result.Password = *values.Password
	}
	if values.Root != nil && !overrides["root"] {
		result.Root = *values.Root
		if !filepath.IsAbs(result.Root) {
			result.Root = filepath.Join(configDir, result.Root)
		}
	}
	if values.ReadOnly != nil && !overrides["readonly"] {
		result.ReadOnly = *values.ReadOnly
	}
	if values.Info != nil && !overrides["info"] {
		result.Info = *values.Info
	}
	if values.URLPrefix != nil && !overrides["url-prefix"] {
		result.URLPrefix = *values.URLPrefix
	}
}

func normalizeURLPrefix(value string) (string, error) {
	if value == "" || value == "/" {
		return "", nil
	}
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\\?#") {
		return "", errors.New("--url-prefix must be an absolute URL path without query, fragment, or backslash")
	}
	value = strings.TrimSuffix(value, "/")
	for _, segment := range strings.Split(strings.TrimPrefix(value, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", errors.New("--url-prefix contains an invalid path segment")
		}
	}
	return value, nil
}

func printUsage(writer io.Writer) {
	fmt.Fprint(writer, `HTTP File Server By iDataDBMS.

Usage:
  http-file-server --root <directory> [--config <file>] [--username <username> --password <password>] [--port <port>] [--url-prefix <path>] [--readonly] [--info] [-h|--help] [--version]

Options:
  --port <port>          HTTP listening port (1-65535, default: 8080)
  --username <username>  HTTP Basic Authentication username (must be used with --password)
  --password <password>  HTTP Basic Authentication password (must be used with --username)
  --root <directory>     Local directory exposed as the HTTP root
  --config <file>        JSON configuration file; command-line options override it

  --url-prefix <path>    URL path prefix for the web page, API, and direct downloads (default: empty)
  --readonly             Allow listing and downloads only
  --info                 Enable HTTP request and diagnostic logs
  -h, --help             Show this help message
  --version              Show version information

Examples:
  http-file-server --username alice --password secret --root D:\file-data
  http-file-server --root D:\file-data
  http-file-server --port 8080 --username viewer --password secret --root D:\file-data --readonly --info
`)
}

func main() {
	configuration, err := parseConfig(os.Args[1:])
	if errors.Is(err, errVersion) {
		fmt.Printf("HTTP File Server %s\n", version)
		return
	}
	if errors.Is(err, flag.ErrHelp) {
		printUsage(os.Stdout)
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	server, err := fileservice.NewServer(configuration.Root, configuration.Username, configuration.Password, configuration.ReadOnly, configuration.URLPrefix)
	if err != nil {
		log.Fatal(err)
	}
	address := fmt.Sprintf(":%d", configuration.Port)
	fmt.Printf("HTTP server configuration:\n  Listening port: %d\n  Storage root: %s\n  URL prefix: %s\n", configuration.Port, configuration.Root, configuration.URLPrefix)
	if configuration.Anonymous {
		if configuration.ReadOnly {
			fmt.Fprintln(os.Stderr, "WARNING: Anonymous access is enabled. Any user can browse and download from this HTTP server.")
		} else {
			fmt.Fprintln(os.Stderr, "WARNING: Anonymous write access is enabled. Any user can browse, upload, move, and delete files.")
		}
	}
	if configuration.Info {
		log.Printf("starting HTTP server (read-only: %t)", configuration.ReadOnly)
	}
	handler := server.Handler()
	if configuration.Info {
		handler = requestLogger(handler)
	}
	log.Fatal(http.ListenAndServe(address, handler))
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *loggingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(response, r)
		if response.status == 0 {
			response.status = http.StatusOK
		}
		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remoteHost = r.RemoteAddr
		}
		log.Printf("%s %s from %s -> %d", r.Method, r.URL.RequestURI(), remoteHost, response.status)
	})
}
