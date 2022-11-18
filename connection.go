package wpa_supplicant

import (
	"context"
	"net"
	"os"
)

// A Connection is a connection to the wpa_supplicant/hostapd control interface socket.
type Connection struct {
	*net.UnixConn
}

func connect(ctrlIfacePath, tempFilePattern string) (*Connection, error) {
	f, err := os.CreateTemp(os.TempDir(), tempFilePattern)
	if err != nil {
		return nil, err
	}

	localPath := f.Name()
	err = os.Remove(localPath)
	if err != nil {
		return nil, err
	}

	unixConn, err := net.DialUnix("unixgram", &net.UnixAddr{
		Name: localPath,
		Net:  "unixgram",
	}, &net.UnixAddr{
		Name: ctrlIfacePath,
		Net:  "unixgram",
	})
	if err != nil {
		return nil, err
	}

	return &Connection{UnixConn: unixConn}, nil
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
func (conn *Connection) SendRequest(ctx context.Context, cmd string) (string, error) {
	_, err := conn.write(ctx, []byte(cmd))
	if err != nil {
		return "", err
	}

	res, err := conn.read(ctx)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

type ioResult struct {
	n   int
	b   []byte
	err error
}

func (conn *Connection) write(ctx context.Context, b []byte) (int, error) {
	ch := make(chan ioResult)

	go func() {
		wrote, err := conn.Write(b)
		ch <- ioResult{
			n:   wrote,
			err: err,
		}
	}()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case result := <-ch:
		{
			return result.n, result.err
		}
	}
}

func (conn *Connection) read(ctx context.Context) ([]byte, error) {
	ch := make(chan ioResult)

	go func() {
		buf := make([]byte, 2048)
		read, err := conn.Read(buf)
		ch <- ioResult{
			n:   read,
			b:   buf,
			err: err,
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		{
			return result.b[0:result.n], result.err
		}
	}
}
