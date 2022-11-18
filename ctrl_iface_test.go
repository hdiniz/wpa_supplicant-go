package wpa_supplicant_test

import (
	"context"
	"errors"
	"github.com/hdiniz/wpa_supplicant-go"
	"github.com/stretchr/testify/require"
	"net"
	"os"
	"testing"
	"time"
)

type mockServer interface {
	Listen(conn *net.UnixConn)
}

type echoServer struct {
	delay time.Duration
}

func (e *echoServer) Listen(conn *net.UnixConn) {
	buf := make([]byte, 2048)
	for {
		read, remote, err := conn.ReadFromUnix(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}

		if e.delay != 0 {
			time.Sleep(e.delay)
		}

		_, err = conn.WriteToUnix(buf[0:read], remote)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
	}
}

type eventServer struct {
	events []string
}

func (e *eventServer) Listen(conn *net.UnixConn) {
	buf := make([]byte, 2048)
	read, remote, err := conn.ReadFromUnix(buf)
	if err != nil {
		return
	}

	if string(buf[0:read]) != "ATTACH" {
		_, err = conn.WriteToUnix([]byte("FAIL\n"), remote)
		if err != nil {
			return
		}
	} else {
		_, err = conn.WriteToUnix([]byte("OK\n"), remote)
		if err != nil {
			return
		}
	}

	for _, event := range e.events {
		_, err = conn.WriteToUnix([]byte(event), remote)
		if err != nil {
			return
		}
	}

	read, remote, err = conn.ReadFromUnix(buf)
	if err != nil {
		return
	}

	if string(buf[0:read]) != "DETACH" {
		_, err = conn.WriteToUnix([]byte("FAIL\n"), remote)
		if err != nil {
			return
		}
	} else {
		_, err = conn.WriteToUnix([]byte("OK\n"), remote)
		if err != nil {
			return
		}
	}
}

func newServer(t *testing.T, mockServer mockServer) string {
	t.Helper()

	f, err := os.CreateTemp(os.TempDir(), "wpa_supplicant-go-test")
	require.NoError(t, err)
	localName := f.Name()

	err = os.Remove(localName)
	require.NoError(t, err)

	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{
		Name: localName,
		Net:  "unixgram",
	})

	require.NoError(t, err)

	t.Cleanup(func() {
		err = conn.Close()
		require.NoError(t, err)
	})

	go mockServer.Listen(conn)
	return localName
}

func TestControlInterface_SendRequest(t *testing.T) {
	ctrlIfacePath := newServer(t, &echoServer{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ctrlIface, err := wpa_supplicant.Connect(ctrlIfacePath)
	require.NoError(t, err)
	defer ctrlIface.Close()

	resp, err := ctrlIface.SendRequest(ctx, "PING")
	require.NoError(t, err)
	require.Equal(t, "PING", resp)

	resp, err = ctrlIface.SendRequest(ctx, "ECHO")
	require.NoError(t, err)
	require.Equal(t, "ECHO", resp)
}

func TestControlInterface_SendRequest_Timeout(t *testing.T) {
	ctrlIfacePath := newServer(t, &echoServer{delay: 50 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ctrlIface, err := wpa_supplicant.Connect(ctrlIfacePath)
	require.NoError(t, err)
	defer ctrlIface.Close()

	newContext, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	_, err = ctrlIface.SendRequest(newContext, "PING")
	require.ErrorContains(t, err, "context deadline exceeded")

	newContext, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	resp, err := ctrlIface.SendRequest(newContext, "ECHO")
	require.NoError(t, err)
	require.Equal(t, "ECHO", resp)
}

func TestControlInterface_Listen(t *testing.T) {
	ctrlIfacePath := newServer(t, &eventServer{events: []string{
		"<1>TEST",
		"<3>SCAN_RESULTS",
		"SCAN",
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ctrlIface, err := wpa_supplicant.Connect(ctrlIfacePath)
	require.NoError(t, err)
	defer ctrlIface.Close()

	listenCtx, cancel := context.WithCancel(ctx)
	var received []wpa_supplicant.Event
	err = ctrlIface.Listen(listenCtx, func(event wpa_supplicant.Event) {
		received = append(received, event)
		if len(received) == 3 {
			cancel()
		}
	})

	require.Len(t, received, 3)
	require.ErrorContains(t, err, "context canceled")

	require.Equal(t,
		[]wpa_supplicant.Event{
			{Priority: wpa_supplicant.EventPriorityDebug, Data: "TEST"},
			{Priority: wpa_supplicant.EventPriorityWarning, Data: "SCAN_RESULTS"},
			{Priority: -1, Data: "SCAN"},
		},
		received,
	)
}

func TestControlInterface_Listen_Timeout(t *testing.T) {
	ctrlIfacePath := newServer(t, &eventServer{})

	ctrlIface, err := wpa_supplicant.Connect(ctrlIfacePath)
	require.NoError(t, err)
	defer ctrlIface.Close()

	listenCtx, cancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
	defer cancel()
	err = ctrlIface.Listen(listenCtx, func(event wpa_supplicant.Event) {
	})

	require.ErrorContains(t, err, "context deadline exceeded")
}
