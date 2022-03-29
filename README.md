# SOCKSForce
*Tool for forcing the connections through a SOCKS5 proxy.*

Intended use: launch the configured SOCKSForce and override the addresses
of the domains in /etc/hosts like so:
```
127.0.0.1 my.domain.name
```

Requests to `my.domain.name` will be received by SOCKSForce, wrapped into a SOCKS5
connection and proxied through specified SOCKS server transparently to other programs.

**NOTE**: listening on privileged ports like 80 or 443 is forbidden for non-root processes
Run `sudo setcap 'cap_net_bind_service=+ep' /path/to/SOCKSForce` to allow it.
If `setcap` is unavailable you have to run `SOCKSForce` as root.

## Installation

`go install github.com/Andrew-Morozko/SOCKSForce@latest`

If someone wants a github release with compiled binaries - create an issue.

## Usage
```
Usage of ./SOCKSForce:
  -config string
        Configuration file (default "./config.json")
```

Fully documented config example can be found in [config_example.json](https://github.com/Andrew-Morozko/SOCKSForce/config_example.json)

Minimal config:
```json
{
    "defaults": {
        "socks": "host/ip[:port]",
    },
    "routes": [
        {
            "listen_port": 80
        },
        {
            "listen_port": 443
        }
    ]
}
```

# Development

Building with `go build -tags enableDebug` enables more debug messages.