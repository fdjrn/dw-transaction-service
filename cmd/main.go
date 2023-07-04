package main

import (
	"fmt"
	"github.com/fdjrn/dw-transaction-service/configs"
	"github.com/fdjrn/dw-transaction-service/internal"
	"github.com/fdjrn/dw-transaction-service/internal/db"
	"github.com/fdjrn/dw-transaction-service/internal/kafka"
	"github.com/fdjrn/dw-transaction-service/internal/routes"
	"github.com/fdjrn/dw-transaction-service/internal/utilities"
	"sync"
)

func main() {
	var err error
	internal.SetupCloseHandler()

	defer internal.ExitGracefully()

	// Service Initialization
	err = configs.Initialize()
	if err != nil {
		utilities.Log.Fatalln(fmt.Sprintf("error on config initialization: %s", err.Error()))
	}

	if err = db.Mongo.Connect(); err != nil {
		utilities.Log.Fatalln(fmt.Sprintf("error on mongodb connection: %s", err.Error()))
	}

	wg := &sync.WaitGroup{}

	// Start Messages Producer
	wg.Add(1)
	go func() {
		err = kafka.Initialize()
		if err != nil {
			utilities.Log.Fatalln(err)
		}
		wg.Done()
	}()

	kafka.StartConsumer()

	// Start Rest API
	wg.Add(1)
	go func() {
		err = routes.Initialize()
		if err != nil {
			utilities.Log.Fatalln(err)
		}

		wg.Done()
	}()

	wg.Wait()

}
