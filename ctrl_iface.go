/*
Package wpa_supplicant provides a control interface to a wpa_supplicant/hostapd daemon.

The Connect function connects to a daemon:
  ctrlIface, err := wpa_supplicant.Connect("/run/wpa_supplicant/wlan0")
  if err != nil {
    // handle err
  }

The SendRequest method sends request to the daemon:

  res, err := ctrlIface.SendRequest(context.TODO(), "PING")
  if err != nil {
    // handle err
  }

  fmt.Println(res)
  // PONG

The control interface can listen to daemon events:

  err := ctrlIface.Listen(context.TODO(), func (event wpa_supplicant.Event) {
    fmt.Println(event)
    // {Priority: wpa_supplicant.EventPriorityInfo, Data: "CTRL-EVENT-SCAN-STARTED"}
  })

  if err != nil {
    // handle err
  }

Refer to wpa_supplicant/hostapd documentation for available requests, responses
and events on this interface.

https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html
*/
package wpa_supplicant

import (
	"context"
	"errors"
	"log"
	"sync"
)

// A ControlInterface is a control interface to a wpa_supplicant/hostapd daemon.
//
// https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html
type ControlInterface struct {
	ctrlIfacePath   string
	tempFilePattern string
	solicitedConn   *Connection
	mutex           sync.Mutex
}

// A ConnectionOptions holds options to construct a ControlInterface.
type ConnectionOptions struct {
	// TemporaryFilePattern is the file name pattern used to generate local socket addresses in os.TempDir().
	TemporaryFilePattern string
}

var (
	DefaultConnectionOptions = ConnectionOptions{
		TemporaryFilePattern: "wpa_supplicant-go-ctrl",
	}
)

// Connect connects to wpa_supplicant/hostapd via the control interface in controlInterfacePath.
func Connect(controlInterfacePath string) (*ControlInterface, error) {
	return ConnectWithOptions(controlInterfacePath, DefaultConnectionOptions)
}

// ConnectWithOptions connects to wpa_supplicant/hostapd via the control interface in controlInterfacePath.
func ConnectWithOptions(controlInterfacePath string, opts ConnectionOptions) (*ControlInterface, error) {
	var ctrlIface ControlInterface
	var err error

	ctrlIface.tempFilePattern = opts.TemporaryFilePattern
	ctrlIface.ctrlIfacePath = controlInterfacePath

	ctrlIface.solicitedConn, err = connect(ctrlIface.ctrlIfacePath, ctrlIface.tempFilePattern)
	if err != nil {
		return nil, err
	}

	return &ctrlIface, err
}

/*
SendRequest sends a request to wpa_supplicant.

  res, err := conn.SendRequest(ctx, "PING")
  if err != nil {
    // handle err
  }

  fmt.Println(res)
  // PONG
*/
func (ctrlIface *ControlInterface) SendRequest(ctx context.Context, cmd string) (string, error) {
	ctrlIface.mutex.Lock()
	defer ctrlIface.mutex.Unlock()
	res, err := ctrlIface.solicitedConn.SendRequest(ctx, cmd)
	if err == nil {
		return res, nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		closeErr := ctrlIface.solicitedConn.Close()
		if closeErr != nil {
			log.Printf("error closing connection: %s\n", closeErr)
		}

		newConn, newConnErr := connect(ctrlIface.ctrlIfacePath, ctrlIface.tempFilePattern)
		if newConnErr != nil {
			if closeErr != nil {
				log.Printf("error opening new connection: %s\n", newConnErr)
			}
			return "", err
		}
		ctrlIface.solicitedConn = newConn
	}

	return "", err
}

// Close closes the connection to wpa_supplicant.
func (ctrlIface *ControlInterface) Close() error {
	return ctrlIface.solicitedConn.Close()
}
