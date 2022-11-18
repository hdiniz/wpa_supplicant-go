package wpa_supplicant

import (
	"context"
	"fmt"
	"log"
	"strconv"
)

// A EventPriority is a value indicating the priority of an event received via the control interface.
//
// https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html
type EventPriority int

const (
	EventPriorityMsgDump = EventPriority(iota)
	EventPriorityDebug
	EventPriorityInfo
	EventPriorityWarning
	EventPriorityError
)

// Event is an unsolicited message from wpa_supplicant.
//
// https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html
type Event struct {
	Priority EventPriority
	Data     string
}

// A UnsolicitedEventHandler handles events received in via the control interface.
type UnsolicitedEventHandler func(Event)

// Listen listens to control interface events.
//
// The function blocks while listening to events, it returns if an
// error occurs while reading the remote connection or if ctx is canceled.
// When an event is received, handler is called on the caller goroutine.
//
// Listen opens a separate connection to handle events only, not interfering with
// the solicited requests/reply exchanges occurring on this ControlInterface.
func (ctrlIface *ControlInterface) Listen(ctx context.Context, handler UnsolicitedEventHandler) error {
	conn, err := connect(ctrlIface.ctrlIfacePath, ctrlIface.tempFilePattern)
	if err != nil {
		return err
	}

	defer conn.Close()

	res, err := conn.SendRequest(ctx, "ATTACH")
	if err != nil {
		return err
	}

	if res != "OK\n" {
		return fmt.Errorf("failed to attach to events: %s", res)
	}

	defer func() {
		_, detachErr := conn.write(context.Background(), []byte("DETACH"))
		if detachErr != nil {
			log.Printf("error sending DETACH request: %s\n", detachErr)
			return
		}
	}()

	var event []byte
	for {
		event, err = conn.read(ctx)
		if err != nil {
			return err
		}

		data := string(event)
		priority := -1
		if len(event) >= 3 && event[0] == '<' && event[2] == '>' {
			priority, _ = strconv.Atoi(string(event[1]))
			data = data[3:]
		}

		handler(Event{
			Priority: EventPriority(priority),
			Data:     data,
		})
	}
}
