# wpa_supplicant-go
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg)](https://pkg.go.dev/github.com/hdiniz/wpa_supplicant-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/hdiniz/wpa_supplicant-go)](https://goreportcard.com/report/github.com/hdiniz/wpa_supplicant-go)
[![codecov](https://codecov.io/github/hdiniz/wpa_supplicant-go/branch/main/graph/badge.svg?token=B7ANCTQGRQ)](https://codecov.io/github/hdiniz/wpa_supplicant-go)
[![License](http://img.shields.io/badge/license-mit-blue.svg)](https://raw.githubusercontent.com/hdiniz/wpa_supplicant-go/master/LICENSE)

Implementation of the wpa_supplicant / hostapd control interface in Go (no CGO).

### Installation

```shell
go get github.com/hdiniz/wpa_supplicant-go
```

### Examples

For complete examples, check the [examples' folder](./examples).

#### [Scanning](./examples/wifi-scan)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"
	
	"github.com/hdiniz/wpa_supplicant-go"
)

func main() {
	ctrl, err := wpa_supplicant.Connect("/run/wpa_supplicant/wlan0")
	if err != nil {
		fmt.Printf("failed to connect to wpa_supplicant: %s\n", err)
		os.Exit(1)
	}
	
	ctx := context.TODO()

	// ask hostapd to start a scan
	res, err := ctrl.SendRequest(ctx, "SCAN")
	if res != "OK\n" {
		fmt.Println("failed to request scan", res)
		os.Exit(1)
	}

	time.Sleep(2 * time.Second) // give some time to scan channels

	res, err = ctrl.SendRequest(ctx, "SCAN_RESULTS")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

#### [Listening to events](./examples/wifi-events)

```go
package main

import (
	"context"
	"fmt"
	"os"
	
	"github.com/hdiniz/wpa_supplicant-go"
)

func main() {
	ctrl, err := wpa_supplicant.Connect("/run/wpa_supplicant/wlan0")
	if err != nil {
		fmt.Printf("failed to connect to wpa_supplicant: %s\n", err)
		os.Exit(1)
	}
	
	ctx := context.TODO()

	err := ctrl.Listen(ctx, func(event wpa_supplicant.Event) {
        fmt.Println(event.Priority, event.Data)
	})
	if err != nil {
		fmt.Printf("failed to listen: %s\n", err)
		os.Exit(1)
	}
}
```

### Permissions

To communicate with the wpa_supplicant/hostapd daemon, the process must have
permission to access the socket path. Likewise, wpa_supplicant/hostapd must have
permissions to access the socket created by this library.

For example, if hostapd is running as a less privileged user (e.g. network)
and the application as root (e.g. sshed into a OpenWRT shell). 
The application will be able to send requests to hostapd, but hostapd will not be 
able to send replies. This can be fixed by running as the same user or by setting
file permissions on the local socket path.