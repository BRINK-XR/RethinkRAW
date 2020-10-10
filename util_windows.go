package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo -64 -icon=assets/favicon.ico -manifest=win.manifest

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	getShortPathName      = kernel32.NewProc("GetShortPathNameW")
	wideCharToMultiByte   = kernel32.NewProc("WideCharToMultiByte")
	getConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	getConsoleWindow      = kernel32.NewProc("GetConsoleWindow")
	showWindow            = user32.NewProc("ShowWindow")
	setForegroundWindow   = user32.NewProc("SetForegroundWindow")
)

func findChrome() string {
	versions := []string{`Google\Chrome`, `Chromium`}
	prefixes := []string{os.Getenv("LOCALAPPDATA"), os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)")}
	suffix := `\Application\chrome.exe`

	for _, v := range versions {
		for _, p := range prefixes {
			if p != "" {
				c := filepath.Join(p, v, suffix)
				if _, err := os.Stat(c); err == nil {
					return c
				}
			}
		}
	}
	return ""
}

func exitChrome(cmd *exec.Cmd) {
	for i := 0; i < 10; i++ {
		if exec.Command("taskkill", "/pid", strconv.Itoa(cmd.Process.Pid)).Run() != nil {
			return
		}
		time.Sleep(time.Second / 10)
	}
}

func openURLCmd(url string) *exec.Cmd {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
}

func isHidden(fi os.FileInfo) bool {
	if strings.HasPrefix(fi.Name(), ".") {
		return true
	}

	if s, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok &&
		s.FileAttributes&(syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM) != 0 {
		return true
	}

	return false
}

func isANSIString(s string) bool {
	if s == "" {
		return true
	}

	var used int32
	long := utf16.Encode([]rune(s))
	n, _, _ := wideCharToMultiByte.Call(0 /*CP_ACP*/, 0x400, /*WC_NO_BEST_FIT_CHARS*/
		uintptr(unsafe.Pointer(&long[0])), uintptr(len(long)), 0, 0, 0,
		uintptr(unsafe.Pointer(&used)))

	return n > 0 && used == 0
}

func getANSIPath(path string) (string, error) {
	path = filepath.Clean(path)

	if len(path) < 260 && isANSIString(path) {
		return path, nil
	}

	vol := len(filepath.VolumeName(path))
	for i := len(path); i >= vol; i-- {
		if i == len(path) || os.IsPathSeparator(path[i]) {
			file := path[:i]
			_, err := os.Stat(file)
			if err == nil {
				if filepath.IsAbs(file) {
					file = `\\?\` + file
				}
				if long, err := syscall.UTF16FromString(file); err == nil {
					short := [264]uint16{}
					n, _, _ := getShortPathName.Call(
						uintptr(unsafe.Pointer(&long[0])),
						uintptr(unsafe.Pointer(&short[0])), 264)
					if 0 < n && n < 264 {
						file = syscall.UTF16ToString(short[:n])
						path = strings.TrimPrefix(file, `\\?\`) + path[i:]
						if len(path) < 260 && isANSIString(path) {
							return path, nil
						}
					}
				}
				break
			}
		}
	}

	return path, errors.New("Could not convert to ANSI path: " + path)
}

func bringToTop() {
	if hwnd, _, _ := getConsoleWindow.Call(); hwnd == 0 {
		return // no window
	} else {
		setForegroundWindow.Call(hwnd)
	}
}

func hideConsole() {
	if hwnd, _, _ := getConsoleWindow.Call(); hwnd == 0 {
		return // no window
	} else {
		var pid uint32
		if n, _, err := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&pid)), 1); n == 0 {
			log.Fatal(err)
		} else if n > 1 {
			return // not the only process
		}
		showWindow.Call(hwnd, 0) // SW_HIDE
	}
}
