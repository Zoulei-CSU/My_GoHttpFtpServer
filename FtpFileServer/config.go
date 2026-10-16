package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type config struct {
	port        int
	username    string
	password    string
	root        string
	readOnly    bool
	info        bool
	anonymous   bool
	pasvMinPort int
	pasvMaxPort int
}

type fileConfig struct {
	Port        *int    `json:"port"`
	Username    *string `json:"username"`
	Password    *string `json:"password"`
	Root        *string `json:"root"`
	ReadOnly    *bool   `json:"readonly"`
	Info        *bool   `json:"info"`
	PasvMinPort *int    `json:"pasv_min_port"`
	PasvMaxPort *int    `json:"pasv_max_port"`
}

func parseConfig(args []string) (config, error) {
	flags := flag.NewFlagSet("ftp-file-server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	result := config{
		port:        2121,
		pasvMinPort: 30000,
		pasvMaxPort: 30100,
	}
	var configPath string
	flags.IntVar(&result.port, "port", 2121, "FTP listening port")
	flags.StringVar(&result.username, "username", "", "FTP username")
	flags.StringVar(&result.password, "password", "", "FTP password")
	flags.StringVar(&result.root, "root", "", "root directory exposed by FTP")
	flags.BoolVar(&result.readOnly, "readonly", false, "disable all write operations")
	flags.BoolVar(&result.info, "info", false, "enable FTP session and diagnostic logs")
	flags.IntVar(&result.pasvMinPort, "pasv-min-port", 30000, "minimum passive-mode data port")
	flags.IntVar(&result.pasvMaxPort, "pasv-max-port", 30100, "maximum passive-mode data port")
	flags.StringVar(&configPath, "config", "", "JSON configuration file")

	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	overrides := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) {
		overrides[item.Name] = true
	})
	if configPath != "" {
		fileValues, configDir, err := readConfigFile(configPath)
		if err != nil {
			return config{}, err
		}
		applyFileConfig(&result, fileValues, configDir, overrides)
	}
	if result.port < 1 || result.port > 65535 {
		return config{}, errors.New("--port must be between 1 and 65535")
	}
	if result.username == "" && result.password == "" {
		result.anonymous = true
	} else if result.username == "" || result.password == "" {
		return config{}, errors.New("--username and --password must be provided together")
	}
	if result.root == "" {
		return config{}, errors.New("--root is required")
	}
	if result.pasvMinPort < 1024 || result.pasvMinPort > 65535 {
		return config{}, errors.New("--pasv-min-port must be between 1024 and 65535")
	}
	if result.pasvMaxPort < 1024 || result.pasvMaxPort > 65535 {
		return config{}, errors.New("--pasv-max-port must be between 1024 and 65535")
	}
	if result.pasvMinPort > result.pasvMaxPort {
		return config{}, errors.New("--pasv-min-port must not exceed --pasv-max-port")
	}

	root, err := filepath.Abs(result.root)
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

	result.root = filepath.Clean(root)
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
		result.port = *values.Port
	}
	if values.Username != nil && !overrides["username"] {
		result.username = *values.Username
	}
	if values.Password != nil && !overrides["password"] {
		result.password = *values.Password
	}
	if values.Root != nil && !overrides["root"] {
		result.root = *values.Root
		if !filepath.IsAbs(result.root) {
			result.root = filepath.Join(configDir, result.root)
		}
	}
	if values.ReadOnly != nil && !overrides["readonly"] {
		result.readOnly = *values.ReadOnly
	}
	if values.Info != nil && !overrides["info"] {
		result.info = *values.Info
	}
	if values.PasvMinPort != nil && !overrides["pasv-min-port"] {
		result.pasvMinPort = *values.PasvMinPort
	}
	if values.PasvMaxPort != nil && !overrides["pasv-max-port"] {
		result.pasvMaxPort = *values.PasvMaxPort
	}
}

func printUsage(writer io.Writer) {
	fmt.Fprint(writer, `FTP File Server By iDataDBMS.

Usage:
  ftp-file-server --root <directory> [--config <file>] [--username <username> --password <password>] [--port <port>] [--readonly] [--info] [--pasv-min-port <port> --pasv-max-port <port>]
  ftp-file-server --version

Options:
  --port <port>          FTP listening port (1-65535, default: 2121)
  --username <username>  FTP login username (must be used with --password)
  --password <password>  FTP login password (must be used with --username)
  --root <directory>     Local directory exposed as the FTP root
  --config <file>        JSON configuration file; command-line options override it
  --readonly             Allow listing and downloads only
  --info                 Enable FTP session and diagnostic logs
  --pasv-min-port <port> Minimum passive-mode data port (default: 30000)
  --pasv-max-port <port> Maximum passive-mode data port (default: 30100)
  --version              Show the program version and exit
  -h, --help             Show this help message

Examples:
  ftp-file-server --username admin --password secret --root D:\ftp-data
  ftp-file-server --root D:\ftp-data
  ftp-file-server --port 2121 --username admin --password secret --root D:\ftp-data --pasv-min-port 30000 --pasv-max-port 30100
  ftp-file-server --username viewer --password secret --root D:\ftp-data --readonly --info
`)
}
