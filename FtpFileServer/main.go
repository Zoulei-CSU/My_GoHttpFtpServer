package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	ftpserver "github.com/goftp/server"
)

func main() {
	if versionRequested(os.Args[1:]) {
		fmt.Println(version)
		return
	}
	if err := run(os.Args[1:], serveUntilInterrupted); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(os.Stdout)
			return
		}
		log.Fatal(err)
	}
}

func run(args []string, serve func(*ftpserver.Server) error) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	fmt.Printf("FTP server configuration:\n  Listening port: %d\n  Storage root: %s\n  Passive ports: %d-%d\n", cfg.port, cfg.root, cfg.pasvMinPort, cfg.pasvMaxPort)
	if cfg.anonymous {
		fmt.Fprintln(os.Stderr, "WARNING: Anonymous access is enabled. Any user can access this FTP server.")
	}
	if cfg.info {
		log.Printf("starting FTP server (read-only: %t)", cfg.readOnly)
	}
	return serve(newFTPServer(cfg))
}

func serveUntilInterrupted(server *ftpserver.Server) error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(done)

	go func() {
		<-done
		if _, ok := server.Logger.(*ftpserver.StdLogger); ok {
			log.Print("shutting down FTP server")
		}
		if err := server.Shutdown(); err != nil {
			log.Printf("FTP server shutdown error: %v", err)
		}
	}()

	err := server.ListenAndServe()
	if err == ftpserver.ErrServerClosed {
		return nil
	}
	return err
}
