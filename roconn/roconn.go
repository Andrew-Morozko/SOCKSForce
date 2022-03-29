package roconn

import (
	"bytes"
	"io"
	"net"
	"time"
)

// Read only net.Conn. Wraps net.Conn and prevents any write/close
// operations from reaching underlying net.Conn.
// All data read from underlying net.Conn is copied into RoConn.Buf
type RoConn struct {
	net.Conn
	Buf bytes.Buffer
}

func (conn *RoConn) Read(p []byte) (n int, err error) {
	n, err = conn.Conn.Read(p)
	if n > 0 {
		_, _ = conn.Buf.Write(p[:n])
	}
	return
}
func (conn *RoConn) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
}
func (conn *RoConn) Close() error {
	return nil
}
func (conn *RoConn) SetDeadline(_ time.Time) error {
	return nil
}
func (conn *RoConn) SetReadDeadline(_ time.Time) error {
	return nil
}
func (conn *RoConn) SetWriteDeadline(_ time.Time) error {
	return nil
}
