//go:build !windows

package main

// What runs differently on a Mac than on Windows (proc_windows.go): tools stopped with the app by a
// shell watchdog and process groups, the viewer site's socket pair and the files handed over it, and
// what's done once the app has gone (Erase All Data, Restart to Update).

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	selfUpdates = true // updates.go: this build can update itself
	onMac       = true // mac.go: the Mac's menus
	// the font the cards are drawn in
	cardFont = "/System/Library/Fonts/Menlo.ttc"
	// MediaMTX and cloudflared are started under this, so they stop when the app
	// does, even if the app is killed or crashes: the app holds the write end of
	// their stdin; when it goes, the watcher's read ends and it stops the tool. (A
	// shell gives background jobs /dev/null for stdin, hence the explicit fd 3.)
	watchdog = `exec 3<&0; "$@" 3<&- & child=$!; (read _ <&3; kill $child 2>/dev/null) & watcher=$!; ` +
		`exec 3<&-; wait $child; code=$?; kill $watcher 2>/dev/null; exit $code`
)

// resourcesDir: Mouse Party.app/Contents/Resources (the pages, the tools); "" if it isn't known
func resourcesDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "..", "Resources")
}

// watched: the command that runs a tool under the watchdog, in a process group of its own
func watched(run string, argv []string) *exec.Cmd {
	cmd := exec.Command("/bin/sh", append([]string{"-c", watchdog, "mouseparty-watch", run}, argv...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// grouped: cmd runs in a process group of its own (ffmpeg: Stop interrupts it and all it started)
func grouped(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// hidden: nothing to hide on a Mac (on Windows, the tool's console window)
func hidden(*exec.Cmd) {}

// started: nothing more to do here once a tool has started (on Windows it joins the app's job)
func started(*exec.Cmd) {}

// interruptGroup: SIGINT, so ffmpeg closes the stream itself
func interruptGroup(cmd *exec.Cmd) { syscall.Kill(-cmd.Process.Pid, syscall.SIGINT) }

func killGroup(cmd *exec.Cmd) { syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }

// stopGroup stops a watched tool and its watcher: they share a process group
func stopGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if syscall.Kill(-pid, syscall.SIGTERM) != nil {
		return
	}
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if syscall.Kill(-pid, 0) != nil {
			return
		}
	}
	syscall.Kill(-pid, syscall.SIGKILL)
}

// killLeftovers: a session that crashed or was force-quit leaves MediaMTX (and
// cloudflared) running, holding the ports. Ours are recognisable: MediaMTX runs a
// mouseparty-host-* config, cloudflared tunnels to our port. Only orphans go (parent
// 1: their app is gone); a live session's belong to its app, which may be another
// copy of this one, or a self-test running beside it.
func (h *Host) killLeftovers() {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,command=").Output()
	if err != nil {
		return
	}
	tunnel := fmt.Sprintf("tunnel --no-autoupdate --url http://127.0.0.1:%d", h.HTTPPort)
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || f[1] != "1" {
			continue
		}
		cmd := strings.Join(f[2:], " ")
		if (strings.Contains(cmd, "mediamtx") && strings.Contains(cmd, "/mouseparty-host-") && strings.HasSuffix(cmd, "mediamtx.yml")) ||
			(strings.Contains(cmd, "cloudflared") && strings.HasSuffix(cmd, tunnel)) {
			h.Log("Stopping a leftover from an earlier session: " + filepath.Base(f[2]))
			var pid int
			fmt.Sscan(f[0], &pid)
			syscall.Kill(pid, syscall.SIGTERM)
		}
	}
	time.Sleep(300 * time.Millisecond)
	// and the folders crashed sessions left (their config; never a key), unless something still uses them
	out, err = exec.Command("ps", "-axo", "command=").Output()
	if err != nil {
		return
	}
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "mouseparty-host-*"))
	for _, d := range dirs {
		if !strings.Contains(string(out), d) {
			os.RemoveAll(d)
		}
	}
}

// socketPair: two ends of a socket, for the app and the site
func socketPair(dgram bool) (ours net.Conn, theirs *os.File, err error) {
	kind := syscall.SOCK_STREAM
	if dgram {
		kind = syscall.SOCK_DGRAM
	}
	syscall.ForkLock.RLock()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, kind, 0)
	if err == nil {
		syscall.CloseOnExec(fds[0])
		syscall.CloseOnExec(fds[1])
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return nil, nil, err
	}
	f := os.NewFile(uintptr(fds[0]), "site")
	ours, err = net.FileConn(f)
	f.Close()
	return ours, os.NewFile(uintptr(fds[1]), "site"), err
}

// sendFile: f, open, to the site, named by token
func sendFile(c *net.UnixConn, token string, f *os.File) error {
	_, _, err := c.WriteMsgUnix([]byte(token), syscall.UnixRights(int(f.Fd())), nil)
	return err
}

// takeFiles: each document's file arrives open, named by the document's token
func (s *siteSide) takeFiles(c *net.UnixConn) {
	name, oob := make([]byte, 256), make([]byte, syscall.CmsgSpace(4*4))
	for {
		n, oobn, _, _, err := c.ReadMsgUnix(name, oob)
		if err != nil {
			return
		}
		msgs, _ := syscall.ParseSocketControlMessage(oob[:oobn])
		for _, m := range msgs {
			fds, _ := syscall.ParseUnixRights(&m)
			for _, fd := range fds {
				syscall.CloseOnExec(fd)
				s.mu.Lock()
				if old := s.handed[string(name[:n])]; old != nil {
					old.Close()
				}
				s.handed[string(name[:n])] = os.NewFile(uintptr(fd), "document")
				s.mu.Unlock()
			}
		}
	}
}

// afterErase: once this process has gone, what it couldn't delete while it ran (its window's
// place and the web view's own storage, which AppKit and WebKit write as it quits), then the app
// opened again, if it's an app (bundle "": wails dev, only the deleting)
func afterErase(bundle string) {
	const id = "io.github.ftrain.mouseparty" // packaging/darwin/Info.plist
	home, _ := os.UserHomeDir()
	lib := filepath.Join(home, "Library")
	var paths []string
	for _, p := range []string{"WebKit/" + id, "Caches/" + id, "HTTPStorages/" + id, "HTTPStorages/" + id + ".binarycookies",
		"Saved Application State/" + id + ".savedState"} {
		paths = append(paths, filepath.Join(lib, p))
	}
	paths = append(paths, erasable()...)
	script := `while kill -0 "$0" 2>/dev/null; do sleep 0.2; done
/usr/bin/defaults delete "$1" 2>/dev/null
shift 2
bundle=$1; shift
rm -rf "$@"
[ -n "$bundle" ] && exec /usr/bin/open "$bundle"`
	cmd := exec.Command("/bin/sh", append([]string{"-c", script, strconv.Itoa(os.Getpid()), id, bundle}, paths...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // not stopped with us
	cmd.Start()
}

// reopen opens the app at path once this process has gone (Restart to Update)
func reopen(path string) {
	cmd := exec.Command("/bin/sh", "-c", `sleep 1; exec /usr/bin/open "$0"`, path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // not stopped with us
	cmd.Start()
}

// afterEraseArg, afterEraseMain: only Windows uses these (proc_windows.go)
const afterEraseArg = "--after-erase"

func afterEraseMain() int { return 2 }
