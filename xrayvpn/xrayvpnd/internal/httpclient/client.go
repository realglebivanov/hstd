package httpclient

import (
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/realglebivanov/hstd/hstdlib"
	"golang.org/x/sys/unix"
)

var Default = defaultClient()
var Direct = directClient()

const timeout = 2 * time.Minute

func defaultClient() *http.Client {
	return &http.Client{Timeout: timeout}
}

func directClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
		Control: directDialerControl,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
	}
}

func directDialerControl(network, address string, c syscall.RawConn) error {
	var seterr error

	setMarkFn := func(fd uintptr) {
		seterr = unix.SetsockoptInt(
			int(fd),
			syscall.SOL_SOCKET,
			syscall.SO_MARK,
			int(hstdlib.XrayOutMark))
	}

	if err := c.Control(setMarkFn); err != nil {
		return err
	}

	return seterr
}
