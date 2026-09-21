//go:build unix

package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func serveLocal(ctx context.Context, socketPath string, handler http.Handler, readHeaderTimeout time.Duration) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0700); err != nil {
		return err
	}
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}()
	if err := os.Chmod(socketPath, 0600); err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: readHeaderTimeout}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
