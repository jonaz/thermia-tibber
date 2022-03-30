package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"github.com/koding/multiconfig"
	"github.com/sirupsen/logrus"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30,
}

var pricesStore = NewPrices()

var wg = &sync.WaitGroup{}

// example run: -token asdf -ip 192.168.10.100 -port 502 -loglevel debug
func main() {
	config := NewConfig()
	multiconfig.MustLoad(config)

	lvl, err := logrus.ParseLevel(config.Loglevel)
	if err != nil {
		log.Println("error setting logrus loglevel: ", err)
		return
	}
	logrus.SetLevel(lvl)

	client := modbus.TCPClient(net.JoinHostPort(config.IP, config.Port))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	wg.Add(2)
	defer stop()
	go tickerPriceFetchLoop(ctx, config)
	go tickerSyncHeapPump(ctx, client)
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

func tickerSyncHeapPump(ctx context.Context, client modbus.Client) {
	defer wg.Done()
	timer := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-timer.C:
			err := syncHeatPump(client)
			if err != nil {
				logrus.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func syncHeatPump(client modbus.Client) error {

	if isSameHourAndDay(time.Now(), pricesStore.cheapestHour) {
		logrus.Info("cheapestHour is now ", pricesStore.cheapestHour)
		_, err := client.WriteSingleRegister(22-1, 5000) // 1000 = 1c
		if err != nil {
			return err
		}

		_, err = client.WriteSingleRegister(23-1, 6000)
		if err != nil {
			return err
		}

		debugCurrentValues(client)
		return nil
	}
	debugCurrentValues(client)
	return nil
}

func debugCurrentValues(client modbus.Client) {
	if logrus.GetLevel() == logrus.DebugLevel {
		debugValue(client, "tart temperature tap water", 22)
		debugValue(client, "Stop temperature tap water", 23)
		debugValue(client, "Tap water weighted temperature", 17)
	}
}

func debugValue(client modbus.Client, text string, address uint16) {
	f, err := ReadInputRegister(client, address)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Debugf("%s: %f", text, f)
}

/*
<nil>
[7 210]
2002
<nil>
[78 32]
20000
<nil>
[15 88]
3928

*/
