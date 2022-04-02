package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

type Destination interface {
	fmt.Stringer
	GetAddr(conn net.Conn) (destinationAddr string, err error)
}

type StaticDestination struct {
	Dest string
}

func (dest *StaticDestination) GetAddr(conn net.Conn) (destinationAddr string, err error) {
	return dest.Dest, nil
}

func (dest *StaticDestination) String() string {
	return dest.Dest
}

type HostHeaderDestination struct {
	ListenPort string
}

func (dest *HostHeaderDestination) GetAddr(conn net.Conn) (destinationAddr string, err error) {
	rq, err := http.ReadRequest(bufio.NewReader(conn))
	host := strings.TrimSpace(rq.Host)
	switch {
	case err != nil:
		err = fmt.Errorf("failed to read host header: %w", err)
	case host == "":
		err = fmt.Errorf("empty host header")
	default:
		destinationAddr = setPort(host, dest.ListenPort)
		if DEBUG {
			log.Printf("HTTP host destination: %s", destinationAddr)
		}
	}
	return
}

func (dest *HostHeaderDestination) String() string {
	return fmt.Sprintf("HTTP_HOST:[%s]", dest.ListenPort)
}

type SNIDestination struct {
	ListenPort string
}

func (dest *SNIDestination) GetAddr(conn net.Conn) (destinationAddr string, err error) {
	var host string
	hostRead := false
	_ = tls.Server(
		conn,
		&tls.Config{
			GetConfigForClient: func(helloInfo *tls.ClientHelloInfo) (*tls.Config, error) {
				host = strings.TrimSpace(helloInfo.ServerName)
				hostRead = true
				return nil, nil
			},
		},
	).Handshake()

	switch {
	case !hostRead:
		err = fmt.Errorf("failed to read TLS SNI")
	case host == "":
		err = fmt.Errorf("empty TLS SNI host name")
	default:
		destinationAddr = setPort(host, dest.ListenPort)
		if DEBUG {
			log.Printf("SNI destination: %s", destinationAddr)
		}
	}
	return
}

func (dest *SNIDestination) String() string {
	return fmt.Sprintf("TLS_SNI:[%s]", dest.ListenPort)
}

func setPort(hostWithOptPort, defaultPort string) string {
	host, port, err := net.SplitHostPort(hostWithOptPort)
	if err != nil {
		host = hostWithOptPort
		port = defaultPort
	}
	return net.JoinHostPort(host, port)
}
