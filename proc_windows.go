package main

// What runs differently on Windows than on a Mac (proc_unix.go). There's no sandbox-exec, so
// sandboxing() is false and the viewer site runs inside the app (host.go: elsewhere), as it does
// with no_sandbox on a Mac: nothing here needs the site's socket pair. Tools are kept from
// outliving the app by a job object that kills everything in it when the app's handle to it
// closes, which Windows does however the app ends. The app doesn't update itself.

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	selfUpdates = false // updates.go: a new version is a new MouseParty.exe
	onMac       = false // mac.go: Windows menus

	// the font the cards are drawn in: Consolas, on every Windows since Vista
	cardFont = `C:\Windows\Fonts\consola.ttf`

	createNoWindow = 0x08000000 // CREATE_NO_WINDOW: a console program shows no console
)

// resourcesDir: the Resources folder beside MouseParty.exe (the pages, the tools); "" if it isn't known
func resourcesDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "Resources")
}

var (
	jobOnce sync.Once
	job     windows.Handle // closed only when the app's process ends, killing what's in it
)

// theJob: a job object that kills whatever is in it once the app has gone
func theJob() windows.Handle {
	jobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "No job object for the tools: "+err.Error())
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
		if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
			fmt.Fprintln(os.Stderr, "No job object for the tools: "+err.Error())
			windows.CloseHandle(h)
			return
		}
		job = h
	})
	return job
}

// hidden: a console program (ffmpeg, MediaMTX, cloudflared) runs without a console window
func hidden(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}

// watched: the command that runs a tool; started puts it in the app's job, so it stops when the app does
func watched(run string, argv []string) *exec.Cmd {
	cmd := exec.Command(run, argv...)
	hidden(cmd)
	return cmd
}

// grouped: ffmpeg, hidden (it joins the job once started)
func grouped(cmd *exec.Cmd) { hidden(cmd) }

// started: the tool joins the app's job, which kills it if the app goes without stopping it
func started(cmd *exec.Cmd) {
	j := theJob()
	if j == 0 || cmd.Process == nil {
		return
	}
	p, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(p)
	windows.AssignProcessToJobObject(j, p)
}

// interruptGroup: Windows can't send ffmpeg a Ctrl+C without a console, so it's stopped outright
// (MediaMTX sees the stream end either way)
func interruptGroup(cmd *exec.Cmd) { cmd.Process.Kill() }

func killGroup(cmd *exec.Cmd) { cmd.Process.Kill() }

// stopGroup stops a watched tool
func stopGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	cmd.Process.Kill()
}

// killLeftovers: tools in the job never outlive the app, so there are none of them; only the
// folders a crashed session left (their config; never a key). One still in use won't delete.
func (h *Host) killLeftovers() {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "mouseparty-host-*"))
	for _, d := range dirs {
		if st, err := os.Stat(d); err == nil && time.Since(st.ModTime()) > time.Minute {
			os.RemoveAll(d)
		}
	}
}

var errNoSite = errors.New("the viewer site runs inside the app on Windows")

func socketPair(bool) (net.Conn, *os.File, error) { return nil, nil, errNoSite }

func sendFile(*net.UnixConn, string, *os.File) error { return errNoSite }

func (s *siteSide) takeFiles(*net.UnixConn) {}

// afterEraseArg: MouseParty.exe started again by Erase All Data, to delete what it couldn't while it ran
const afterEraseArg = "--after-erase"

// afterErase: once this process has gone, what it couldn't delete while it ran (the web view's own
// storage, which WebView2 holds open until the app quits), then the app opened again
func afterErase(string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	paths := erasable()
	if dir, err := os.UserConfigDir(); err == nil { // where Wails keeps WebView2's data: %APPDATA%\MouseParty.exe
		paths = append(paths, filepath.Join(dir, filepath.Base(exe)))
	}
	cmd := exec.Command(exe, append([]string{afterEraseArg, strconv.Itoa(os.Getpid())}, paths...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS}
	cmd.Start()
}

// afterEraseMain: waits for the app (its pid) to be gone, deletes the paths, and opens it again
func afterEraseMain() int {
	if len(os.Args) < 3 {
		return 2
	}
	if pid, err := strconv.Atoi(os.Args[2]); err == nil {
		if p, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)); err == nil {
			windows.WaitForSingleObject(p, 30_000)
			windows.CloseHandle(p)
		}
	}
	for _, path := range os.Args[3:] {
		for try := 0; try < 20; try++ { // WebView2's own processes let go of its folder a moment later
			if os.RemoveAll(path) == nil {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
	reopen("")
	return 0
}

// reopen opens MouseParty.exe again, on its own (not stopped with this process)
func reopen(string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS}
	cmd.Start()
}
