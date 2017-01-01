package netx

import (
	"errors"
	"net"
)

func originalTargetAddr(conn net.Conn) (net.Addr, error) {
	return nil, errors.New("netx.OriginalTargetAddr is not implemented on windows")
}
