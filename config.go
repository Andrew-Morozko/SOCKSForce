package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"muzzammil.xyz/jsonc"
)

type JSONConfig struct {
	Defaults struct {
		Socks          string `json:"socks"`
		ListenIp       string `json:"listen_ip"`
		Destination    string `json:"destination"`
		ConnectTimeout string `json:"connect_timeout"`
		ProxyTimeout   string `json:"proxy_timeout"`
	} `json:"defaults"`
	Routes []struct {
		Socks          string      `json:"socks"`
		ListenIP       string      `json:"listen_ip"`
		ListenPort     interface{} `json:"listen_port,omitempty"`
		Destination    string      `json:"destination"`
		ConnectTimeout string      `json:"connect_timeout"`
		ProxyTimeout   string      `json:"proxy_timeout"`
	} `json:"routes"`
}

type RouteConfig struct {
	Num            int
	Socks          string
	ListenAddr     string
	Destination    Destination
	ProxyTimeout   time.Duration
	ConnectTimeout time.Duration
}

func (r *RouteConfig) toString(dest string) string {
	if dest == "" {
		dest = r.Destination.String()
	}
	return fmt.Sprintf("Route #%d: %s <-SOCKS-> %s <-> %s", r.Num, r.ListenAddr, r.Socks, dest)
}

func (r *RouteConfig) String() string {
	return r.toString(r.Destination.String())
}

func readConfig() (*JSONConfig, error) {
	configFile := flag.String("config", "./config.json", "Configuration file")
	flag.Parse()

	var config JSONConfig
	f, err := os.Open(*configFile)
	if err != nil {
		return nil, fmt.Errorf(`can't open config file "%s": %w`, *configFile, err)
	}
	defer f.Close()
	configTest, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf(`can't read config file "%s": %w`, *configFile, err)
	}
	err = jsonc.Unmarshal(configTest, &config)
	if err != nil {
		return nil, fmt.Errorf(`can't parse config file "%s": %w`, *configFile, err)
	}
	return &config, nil
}

func parseConfig() (routes []*RouteConfig, err error) {
	config, err := readConfig()
	if err != nil {
		return
	}

	if len(config.Routes) == 0 {
		err = fmt.Errorf(`no routes specified`)
		return
	}
	if config.Defaults.ListenIp == "" {
		config.Defaults.ListenIp = "127.0.0.1"
	}

	defaultProxyTimeout := time.Duration(0)
	defaultConnectTimeout := 10 * time.Second

	if config.Defaults.ProxyTimeout != "" {
		defaultProxyTimeout, err = time.ParseDuration(config.Defaults.ProxyTimeout)
		if err != nil {
			err = fmt.Errorf("incorrect default proxy_timeout value: %s", err)
			return
		}
	}
	if config.Defaults.ConnectTimeout != "" {
		defaultConnectTimeout, err = time.ParseDuration(config.Defaults.ConnectTimeout)
		if err != nil {
			err = fmt.Errorf("incorrect default connect_timeout value: %s", err)
			return
		}
	}

	for i, jsonRoute := range config.Routes {
		route := new(RouteConfig)
		route.Num = i + 1
		if jsonRoute.Socks == "" {
			jsonRoute.Socks = config.Defaults.Socks
		}
		if jsonRoute.ListenIP == "" {
			jsonRoute.ListenIP = config.Defaults.ListenIp
		}

		switch {
		case jsonRoute.Socks == "":
			err = fmt.Errorf("route #%d: missing SOCKS server address", route.Num)
			return
		case jsonRoute.ListenIP == "":
			err = fmt.Errorf("route #%d: missing listen IP address", route.Num)
			return
		case jsonRoute.ListenPort == nil:
			err = fmt.Errorf("route #%d: missing listen port", route.Num)
			return
		}
		route.Socks = setPort(jsonRoute.Socks, "1080")

		if jsonRoute.ProxyTimeout != "" {
			route.ProxyTimeout, err = time.ParseDuration(jsonRoute.ProxyTimeout)
			if err != nil {
				err = fmt.Errorf("route #%d: incorrect proxy_timeout value: %s", route.Num, err)
				return
			}
		} else {
			route.ProxyTimeout = defaultProxyTimeout
		}
		if jsonRoute.ConnectTimeout != "" {
			route.ConnectTimeout, err = time.ParseDuration(jsonRoute.ConnectTimeout)
			if err != nil {
				err = fmt.Errorf("route #%d: incorrect connect_timeout value: %s", route.Num, err)
				return
			}
		} else {
			route.ConnectTimeout = defaultConnectTimeout
		}

		var listenPort string
		switch port := jsonRoute.ListenPort.(type) {
		case float64:
			listenPort = fmt.Sprintf("%.0f", port)
		case string:
			listenPort = port
		default:
			err = fmt.Errorf("route #%d: incorrect listen port value", route.Num)
			return
		}
		route.ListenAddr = net.JoinHostPort(jsonRoute.ListenIP, listenPort)

		if jsonRoute.Destination == "" {
			switch listenPort {
			case "80":
				jsonRoute.Destination = "http_host"
			case "443":
				jsonRoute.Destination = "tls_sni"
			default:
				if config.Defaults.Destination == "" {
					err = fmt.Errorf("route #%d: missing destination identification method", route.Num)
					return
				}
				jsonRoute.Destination = config.Defaults.Destination
			}
		}

		jsonRoute.Destination = strings.TrimSpace(jsonRoute.Destination)
		switch strings.ToLower(jsonRoute.Destination) {
		case "http_host":
			route.Destination = &HostHeaderDestination{
				ListenPort: listenPort,
			}
		case "tls_sni":
			route.Destination = &SNIDestination{
				ListenPort: listenPort,
			}
		default:
			route.Destination = &StaticDestination{
				Dest: setPort(jsonRoute.Destination, listenPort),
			}
		}
		routes = append(routes, route)
	}

	if DEBUG {
		log.Println("Configuration is successfully parsed:")
		for _, route := range routes {
			log.Print(route)
		}
	}
	err = nil
	return
}
