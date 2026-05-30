//go:build client && !windows

package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// newIPCTransport returns an http.Transport that connects via Unix socket.
func newIPCTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, 5*time.Second)
		},
	}
}

// knownIPCPath returns the well-known Unix socket path for Mihomo.
// Clash Verge Rev uses <app_home_dir>/verge/verge-mihomo.sock
func knownIPCPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	socketPath := filepath.Join(home, ".config", "io.github.clash-verge-rev.clash-verge-rev", "verge", "verge-mihomo.sock")
	if _, err := os.Stat(socketPath); err == nil {
		return socketPath
	}
	return ""
}
