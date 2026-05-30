//go:build client && windows

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32     = windows.NewLazySystemDLL("kernel32.dll")
	procCreateFileW = modkernel32.NewProc("CreateFileW")
)

const (
	_PIPE_ACCESS_DUPLEX   = 0x00000003
	_INVALID_HANDLE_VALUE = ^uintptr(0)
	_ERROR_PIPE_BUSY      = 231
)

// npipeConn wraps a Windows named pipe handle as net.Conn.
type npipeConn struct {
	handle windows.Handle
}

func (c *npipeConn) Read(b []byte) (int, error) {
	var n uint32
	err := windows.ReadFile(c.handle, b, &n, nil)
	return int(n), err
}

func (c *npipeConn) Write(b []byte) (int, error) {
	var n uint32
	err := windows.WriteFile(c.handle, b, &n, nil)
	return int(n), err
}

func (c *npipeConn) Close() error                       { return windows.CloseHandle(c.handle) }
func (c *npipeConn) LocalAddr() net.Addr                { return npipeAddr{} }
func (c *npipeConn) RemoteAddr() net.Addr               { return npipeAddr{} }
func (c *npipeConn) SetDeadline(t time.Time) error      { return nil }
func (c *npipeConn) SetReadDeadline(t time.Time) error   { return nil }
func (c *npipeConn) SetWriteDeadline(t time.Time) error  { return nil }

type npipeAddr struct{}

func (npipeAddr) Network() string { return "npipe" }
func (npipeAddr) String() string  { return "named-pipe" }

func dialNamedPipe(path string, timeout time.Duration) (net.Conn, error) {
	pathUTF16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("invalid pipe path: %w", err)
	}

	deadline := time.Now().Add(timeout)

	for {
		handle, _, callErr := procCreateFileW.Call(
			uintptr(unsafe.Pointer(pathUTF16)),
			_PIPE_ACCESS_DUPLEX,
			0, 0, 3, 0, 0,
		)
		if handle != _INVALID_HANDLE_VALUE {
			return &npipeConn{handle: windows.Handle(handle)}, nil
		}
		if callErr != windows.Errno(_ERROR_PIPE_BUSY) {
			return nil, fmt.Errorf("open pipe %s: %w", path, callErr)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("pipe %s busy timeout", path)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// newIPCTransport returns an http.Transport that connects via Windows named pipe.
func newIPCTransport(pipePath string) *http.Transport {
	return &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return dialNamedPipe(pipePath, 5*time.Second)
		},
	}
}

// knownIPCPath returns the well-known Windows named pipe path for Mihomo.
func knownIPCPath() string {
	return `\\.\pipe\verge-mihomo`
}
