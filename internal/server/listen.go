package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// Listen binds addr. When the port is taken it reports the owning process
// if it can be found, and never falls back to another port.
func Listen(host string, port int) (net.Listener, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		return ln, nil
	}
	if !errors.Is(err, syscall.EADDRINUSE) {
		return nil, err
	}
	msg := fmt.Sprintf("port %d is already in use", port)
	if pid := portOwner(port); pid != "" {
		msg += fmt.Sprintf(" by process %s", pid)
	}
	return nil, fmt.Errorf("%s; stop that process or pass --port", msg)
}

// portOwner finds the PID listening on a TCP port, best effort.
func portOwner(port int) string {
	if runtime.GOOS == "linux" {
		if pid := procPortOwner(port); pid != "" {
			return pid
		}
	}
	if _, err := exec.LookPath("lsof"); err == nil {
		out, err := exec.Command("lsof", "-nP", "-t", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN").Output()
		if err == nil {
			if pid := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0]); pid != "" {
				return pid
			}
		}
	}
	return ""
}

// procPortOwner maps the listening socket inode from /proc/net/tcp{,6} to a
// process by scanning /proc/<pid>/fd. Only processes of the current user are
// readable, which covers the usual case of a forgotten muninn serve.
func procPortOwner(port int) string {
	inodes := map[string]bool{}
	for _, f := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		file, err := os.Open(f)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(file)
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			// local_address is HEXIP:HEXPORT; state 0A is LISTEN.
			if len(fields) < 10 || fields[3] != "0A" {
				continue
			}
			_, hexPort, ok := strings.Cut(fields[1], ":")
			if p, err := strconv.ParseInt(hexPort, 16, 32); ok && err == nil && int(p) == port {
				inodes[fields[9]] = true
			}
		}
		file.Close()
	}
	if len(inodes) == 0 {
		return ""
	}
	fds, _ := filepath.Glob("/proc/[0-9]*/fd/*")
	for _, fd := range fds {
		link, err := os.Readlink(fd)
		if err != nil || !strings.HasPrefix(link, "socket:[") {
			continue
		}
		if inodes[strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")] {
			return strings.Split(fd, "/")[2]
		}
	}
	return ""
}

// IsLoopback reports whether host only accepts local connections.
func IsLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
