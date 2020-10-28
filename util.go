package main

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)
const MinInt = -MaxInt - 1

type constError string

func (e constError) Error() string { return string(e) }

func init() {
	must(mime.AddExtensionType(".dng", "image/x-adobe-dng"))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func hash(data string) string {
	h := md5.Sum([]byte(data))
	return base64.URLEncoding.EncodeToString(h[:15])
}

func index(a []string, x string) int {
	for k, v := range a {
		if x == v {
			return k
		}
	}
	return -1
}

func unique(a *[]string) {
	s := *a
	if len(s) < 1 {
		return
	}

	sort.Strings(s)

	i := 0
	for j := 1; j < len(s); j++ {
		if s[i] != s[j] {
			i++
			s[i] = s[j]
		}
	}
	i++

	*a = s[:i:i]
}

func toASCII(str string) string {
	builder := strings.Builder{}
	for _, r := range str {
		// control
		if r <= 0x1f || 0x7f <= r && r <= 0x9f {
			continue
		}
		// unicode
		if r >= 0xa0 {
			builder.WriteByte('?')
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func filename(name string) string {
	builder := strings.Builder{}
	dots := 0

	for _, r := range name {
		switch r {
		case '\\', '/', ':', '*', '?', '<', '>', '|':
			// Windows doesn't like these.
		case '"':
			builder.WriteByte('\'')
		case '.':
			builder.WriteByte('.')
			dots += 1
		default:
			if strconv.IsPrint(r) {
				builder.WriteRune(r)
			}
		}
	}

	if builder.Len() > dots {
		return builder.String()
	}
	return ""
}

var newFilenameRE = regexp.MustCompile(`\A(.*?)(?: \((\d{1,4})\))?(\.\w*)?\z`)

func makeFile(name string) (*os.File, error) {
	for {
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if os.IsExist(err) {
			m := newFilenameRE.FindStringSubmatch(name)
			if m != nil {
				var i = 0
				if m[2] != "" {
					i, _ = strconv.Atoi(m[2])
				}
				name = m[1] + " (" + strconv.Itoa(i+1) + ")" + m[3]
				continue
			}
		}
		return f, err
	}
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}

func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if isNotSameDevice(err) {
		if err := copyFile(src, dst); err != nil {
			return err
		}
		if err := os.Remove(src); os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	return err
}

func lnkyFile(src, dst string) error {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}

	dfi, _ := os.Stat(dst)
	if os.SameFile(sfi, dfi) {
		return nil
	}

	err = os.Link(src, dst)
	if isNotSameDevice(err) {
		return copyFile(src, dst)
	}
	return err
}

func isNotSameDevice(err error) bool {
	var lerr *os.LinkError
	if errors.As(err, &lerr) {
		if runtime.GOOS == "windows" {
			return lerr.Err == syscall.Errno(0x11) // ERROR_NOT_SAME_DEVICE
		} else {
			return lerr.Err == syscall.Errno(0x12) // EXDEV
		}
	}
	return false
}
