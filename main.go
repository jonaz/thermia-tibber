package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/koding/multiconfig"
	"github.com/sirupsen/logrus"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30,
}

var pricesStore = NewPrices()

var wg = &sync.WaitGroup{}
var fileLogger = logrus.New()

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

	file, err := os.OpenFile(config.LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		logrus.Fatal(err)
	}
	defer file.Close()
	fileLogger.SetOutput(file)

	r := gin.Default()
	r.StaticFile("/", config.LogFile)
	go func() {
		err := r.Run()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Error(err)
		}
	}()

	client := modbus.TCPClient(net.JoinHostPort(config.IP, config.Port))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	wg.Add(2)
	defer stop()
	go tickerPriceFetchLoop(ctx, config)
	go tickerSyncHeapPump(ctx, client, config)
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

func tickerSyncHeapPump(ctx context.Context, client modbus.Client, config *Config) {
	defer wg.Done()
	timer := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-timer.C:
			err := syncHeatPump(client, config)
			if err != nil {
				logrus.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func syncHeatPump(client modbus.Client, config *Config) error {

	if isSameHourAndDay(time.Now(), pricesStore.cheapestHour) {
		logrus.Info("cheapestHour is now ", pricesStore.cheapestHour)
		err := writeTemps(client, config.CheapStartTemp, config.CheapStopTemp)
		if err != nil {
			return err
		}

		debugCurrentValues(client)
		return nil
	}

	debugCurrentValues(client)
	// set back default values
	err := writeTemps(client, config.CheapStartTemp, config.CheapStopTemp)
	if err != nil {
		return err
	}
	return nil
}

func writeTemps(client modbus.Client, start, stop int) error {

	_, err := client.WriteSingleRegister(22, uint16(start*100)) // 1000 = 1c
	if err != nil {
		return err
	}

	_, err = client.WriteSingleRegister(23, uint16(stop*100))
	if err != nil {
		return err
	}
	return nil
}

func debugCurrentValues(client modbus.Client) {
	if logrus.GetLevel() == logrus.DebugLevel {
		debugHoldingValue(client, "Start temp tap water", 22)
		debugHoldingValue(client, "Stop temp tap water", 23)
		debugInputValue(client, "Tap water weighted temperature", 17)
	}
}

func debugHoldingValue(client modbus.Client, text string, address uint16) {
	f, err := ReadHoldingRegister(client, address)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Debugf("%s: %f", text, f)
}
func debugInputValue(client modbus.Client, text string, address uint16) {
	f, err := ReadInputRegister(client, address)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Debugf("%s: %f", text, f)
}
