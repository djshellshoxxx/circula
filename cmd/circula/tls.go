package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

// Native TLS (RB-013). When tls-cert-file and tls-key-file are set the process
// terminates TLS itself: minimum TLS 1.2, Go's default (modern) cipher suites,
// and the certificate pair is re-read on SIGHUP so renewals need no restart.
// A failed reload keeps the previous certificate.

// certReloader serves the current certificate pair.
type certReloader struct {
	certFile, keyFile string
	mu                sync.RWMutex
	cert              *tls.Certificate
}

func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	r := &certReloader{certFile: certFile, keyFile: keyFile}
	if err := r.reload(); err != nil {
		return nil, err
	}
	return r, nil
}

// reload reads the pair from disk. The key file must not be readable by group
// or others (Unix).
func (r *certReloader) reload() error {
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(r.keyFile)
		if err != nil {
			return fmt.Errorf("tls-key-file: %w", err)
		}
		if fi.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("tls-key-file %s is readable by group or others (mode %o); chmod 600 it", r.keyFile, fi.Mode().Perm())
		}
	}
	c, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return fmt.Errorf("load tls certificate: %w", err)
	}
	r.mu.Lock()
	r.cert = &c
	r.mu.Unlock()
	return nil
}

func (r *certReloader) getCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cert, nil
}

// tlsConfig returns the server configuration for the reloader.
func (r *certReloader) tlsConfig() *tls.Config {
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: r.getCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
	}
}

// reloadOnSIGHUP re-reads the pair on every SIGHUP until the returned stop is called.
func (r *certReloader) reloadOnSIGHUP(logger *slog.Logger) (stop func()) {
	ch := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(ch, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-ch:
				if err := r.reload(); err != nil {
					logger.Error("tls certificate reload failed; keeping the previous certificate", "error", err)
				} else {
					logger.Info("tls certificate reloaded")
				}
			case <-done:
				return
			}
		}
	}()
	return func() { signal.Stop(ch); close(done) }
}

// tlsEnabled validates the tls-* settings: both or neither, and an https
// public base URL when enabled.
func tlsEnabled(cfg *Config) (bool, error) {
	if cfg.TLSCertFile == "" && cfg.TLSKeyFile == "" {
		return false, nil
	}
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		return false, errors.New("tls-cert-file and tls-key-file must be set together")
	}
	if !strings.HasPrefix(cfg.PublicBaseURL, "https://") {
		return false, errors.New("tls-cert-file is set, so public-base-url must be https://")
	}
	return true, nil
}
