package internal

import (
	"github.com/fdjrn/dw-transaction-service/internal/db"
	"github.com/fdjrn/dw-transaction-service/internal/kafka"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func ExitGracefully() {
	// close mongodb connection
	utilities.Log.SetPrefix("[EXIT-APP] ")
	if err := db.Mongo.Disconnect(); err != nil {
		log.Println(err.Error())
		return
	}
	utilities.Log.Println("| db connection successfully closed")

	// close kafka connection
	_ = kafka.Producer.Close()
	utilities.Log.Println("| kafka producer successfully closed")
}

// SetupCloseHandler :
func SetupCloseHandler() {

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		utilities.Log.SetPrefix("[EXIT-APP] ")
		utilities.Log.Println("| Ctrl+C pressed in Terminal,... Good Bye...")
		ExitGracefully()
		os.Exit(0)
	}()
}
