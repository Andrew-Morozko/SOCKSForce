package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Andrew-Morozko/SOCKSForce/roconn"
	"golang.org/x/net/proxy"
)

func doProxying(clientConn net.Conn, route *RouteConfig, destAddr string, buf *bytes.Buffer) (err error) {
	var localWG sync.WaitGroup

	proxyDialer, err := proxy.SOCKS5("tcp", route.Socks, nil, &net.Dialer{Timeout: route.ConnectTimeout})
	if err != nil {
		return fmt.Errorf("failed to connect to SOCKS server: %s", err)
	}

	destConn, err := proxyDialer.Dial("tcp", destAddr)
	if err != nil {
		return fmt.Errorf("failed to connect through SOCKS server: %s", err)
	}

	defer destConn.Close()

	localWG.Add(2)
	go func() {
		defer func() {
			_ = clientConn.(*net.TCPConn).CloseWrite()
			localWG.Done()
		}()
		_, _ = io.Copy(clientConn, destConn)
	}()
	go func() {
		defer func() {
			_ = destConn.(*net.TCPConn).CloseWrite()
			localWG.Done()
		}()
		if buf != nil {
			_, err := io.Copy(destConn, buf)
			buf = nil
			if err != nil {
				return
			}
		}
		_, _ = io.Copy(destConn, clientConn)
	}()

	localWG.Wait()
	return
}

func handleConnection(wg *sync.WaitGroup, route *RouteConfig, conn net.Conn) {
	defer wg.Done()
	var destAddr string
	// close connection after timeout
	var timer *time.Timer
	if route.ProxyTimeout != 0 {
		timer = time.AfterFunc(route.ProxyTimeout, func() {
			if DEBUG {
				log.Printf("Finished (timeout): %s", route.toString(destAddr))
			}
			conn.Close()
		})
	}

	defer func() {
		if timer == nil || timer.Stop() {
			// exiting function before the connection timed-out, closing it
			if DEBUG {
				log.Printf("Finished: %s", route.toString(destAddr))
			}
			conn.Close()
		}

	}()

	roConn := &roconn.RoConn{
		Conn: conn,
	}
	var err error
	destAddr, err = route.Destination.GetAddr(roConn)
	if err != nil {
		if DEBUG {
			log.Printf("%s\nFailed to determine destination address: %s", route, err)
		} else {
			log.Printf("Route #%d: failed to determine destination address: %s", route.Num, err)
		}
		return
	}
	var buf *bytes.Buffer
	if roConn.Buf.Len() != 0 {
		buf = &roConn.Buf
	}
	roConn = nil

	if DEBUG {
		log.Printf("Started: %s", route.toString(destAddr))
	}

	err = doProxying(conn, route, destAddr, buf)
	if err != nil {
		if DEBUG {
			log.Printf("%s failed: %s", route.toString(destAddr), err)
		} else {
			log.Printf("Route #%d: proxying failed: %s", route.Num, err)
		}
		return
	}
}

type handlerFunc func(net.Conn)

func acceptAndHandle(wg *sync.WaitGroup, l net.Listener, route *RouteConfig) {
	defer wg.Done()
	for {
		conn, err := l.Accept()
		if err != nil {
			// probably closed listener sock
			return
		}
		wg.Add(1)
		go handleConnection(wg, route, conn)
	}
}

var forcedShutdown = errors.New("forced shutdown")

func multiProxyServer(routes []*RouteConfig) (err error) {
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)
	wg := new(sync.WaitGroup)

	listeners := make([]net.Listener, 0, len(routes))

	defer func() {
		for _, l := range listeners {
			l.Close()
		}
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-signalC:
			err = forcedShutdown
		}
	}()

	for _, route := range routes {
		listener, err := net.Listen("tcp", route.ListenAddr)
		if err != nil {
			return fmt.Errorf("route #%d: failed to start listening on %s: %s", route.Num, route.ListenAddr, err)
		}
		listeners = append(listeners, listener)
		wg.Add(1)
		go acceptAndHandle(wg, listener, route)
	}

	<-signalC
	log.Println("Shutdown requested, finalizing connections")
	err = nil
	return
}

func main() {
	if DEBUG {
		log.Println("Parsing config...")
	}

	routes, err := parseConfig()
	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	log.Println("Starting proxy server...")
	err = multiProxyServer(routes)
	switch err {
	case nil:
		log.Println("Proxy server finished")
	case forcedShutdown:
		log.Fatal("Forced server shutdown, dropping client connections")
	default:
		log.Fatalf("Proxy error: %s", err)
	}
}
