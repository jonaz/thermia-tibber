package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/koding/multiconfig"
	"github.com/sirupsen/logrus"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30,
}

var pricesStore = NewPrices()

var wg = &sync.WaitGroup{}

func main() {
	config := NewConfig()
	multiconfig.MustLoad(config)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	wg.Add(2)
	defer stop()
	go tickerPriceFetchLoop(ctx, config)
	go tickerSyncHeapPump(ctx)
	wg.Wait()
}

func tickerPriceFetchLoop(ctx context.Context, config *Config) {
	defer wg.Done()
	delay := nextDelay()
	timer := time.NewTimer(delay)
	fetchAndCalculate(config)
	logrus.Info("scheduling first run in", delay)
	for {
		select {
		case <-timer.C:
			timer.Reset(nextDelay())
			fetchAndCalculate(config)
		case <-ctx.Done():
			return
		}
	}
}

func tickerSyncHeapPump(ctx context.Context) {
	defer wg.Done()
	timer := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-timer.C:
			err := syncHeatPump()
			if err != nil {
				logrus.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func syncHeatPump() error {
	if isSameHourAndDay(time.Now(), pricesStore.cheapestHour) {
		logrus.Info("cheapestHour is now ", pricesStore.cheapestHour)
		//TODO modbus code here to talk to therma and make more hotwater!
	}
	return nil
}
