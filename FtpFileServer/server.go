package main

import (
	"fmt"

	ftpserver "github.com/goftp/server"
)

func newFTPServer(cfg config) *ftpserver.Server {
	logger := ftpserver.Logger(&ftpserver.DiscardLogger{})
	if cfg.info {
		logger = &ftpserver.StdLogger{}
	}
	owner := cfg.username
	auth := ftpserver.Auth(&ftpserver.SimpleAuth{Name: cfg.username, Password: cfg.password})
	if cfg.anonymous {
		owner = "anonymous"
		auth = anonymousAuth{}
	}

	return ftpserver.NewServer(&ftpserver.ServerOpts{
		Factory: fileDriverFactory{
			root:     cfg.root,
			readOnly: cfg.readOnly,
			owner:    owner,
		},
		Auth:     auth,
		Name:     "FTP File Server By iDataDBMS.",
		Hostname: "0.0.0.0",
		Port:     cfg.port,
		Logger:   logger,
		// The dependency treats its upper bound as exclusive; preserve the
		// inclusive command-line range used by the firewall configuration.
		PassivePorts: fmt.Sprintf("%d-%d", cfg.pasvMinPort, cfg.pasvMaxPort+1),
	})
}

type anonymousAuth struct{}

func (anonymousAuth) CheckPasswd(string, string) (bool, error) {
	return true, nil
}
