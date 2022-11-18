package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/hdiniz/wpa_supplicant-go"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: wifi-events <wpa_supplicant control interface path>")
		os.Exit(1)
	}

	ctrlIfacePath := os.Args[1]

	ctrl, err := wpa_supplicant.Connect(ctrlIfacePath)
	if err != nil {
		fmt.Printf("failed to connect to wpa_supplicant: %s\n", err)
		os.Exit(1)
	}

	defer ctrl.Close()

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt)

	ctx, cancelCtx := context.WithTimeout(context.Background(), 60*time.Second)

	go func() {
		<-exitCh
		cancelCtx()
		time.Sleep(3 * time.Second)
		os.Exit(1)
	}()

	err = ctrl.Listen(ctx, func(event wpa_supplicant.Event) {
		fmt.Printf("%d - %s\n", event.Priority, event.Data)
	})

	if err != nil {
		fmt.Println(err)
	}
}
