package util

import (
	"net"
	"strings"
	"os"
)

// StringBetween gets the string in between two other strings, and returns an empty string if not found. It returns the first match.
func StringBetween(str, start, end string) string {
	s := strings.Index(str, start)
	if s == -1 {
		return ""
	}
	s += len(start)
	if s >= len(str) {
		return ""
	}
	e := strings.Index(str, end)
	if e == -1 {
		return ""
	}
	return str[s:e]
}

// StringAfter gets the string after another.
func StringAfter(str, start string) string {
	s := strings.Index(str, start)
	if s == -1 {
		return ""
	}
	s += len(start)
	if s >= len(str) {
		return ""
	}
	return str[s:]
}

// FixString fixes some issues with strings in metadata.
func FixString(s string) string {
	return strings.Map(func(in rune) rune {
		switch in {
		case '“', '‹', '”', '›':
			return '"'
		case '‘', '’':
			return '\''
		}
		return in
	}, s)
}

// GetIP gets the preferred outbound ip of this machine.
func GetIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func DirExists(pathname string) bool {
	stat, err := os.Stat(pathname)
	if os.IsNotExist(err)  {
		return false
	}
	return stat.IsDir()
}
