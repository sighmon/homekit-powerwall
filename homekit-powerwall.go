package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/foogod/go-powerwall"

	"github.com/sighmon/homekit-powerwall/powerwall_2"
	"github.com/sighmon/homekit-powerwall/promexporter"
)

var acc *accessory.Thermometer
var prometheusExporter bool
var powerwallPrometheusExporter *promexporter.Exporter
var timeBetweenReadings int
var powerwallIP string
var username string
var password string

const inputDefault = ""

func init() {
	flag.StringVar(&powerwallIP, "ip", inputDefault, "IP address of Powerwall")
	flag.StringVar(&username, "username", inputDefault, "Username setup on your Powerwall")
	flag.StringVar(&password, "password", inputDefault, "Password setup on your Powerwall")
	flag.BoolVar(&prometheusExporter, "prometheusExporter", false, "Start a Prometheus exporter on port 8000")
	flag.IntVar(&timeBetweenReadings, "timeBetweenReadings", 30, "The time in seconds between Powerwall readings")
	flag.Parse()
}

func startPrometheus() {
	powerwallPrometheusExporter = promexporter.New(":8000")
	powerwallPrometheusExporter.Start()
}

func main() {
	if powerwallIP == inputDefault || username == inputDefault || password == inputDefault {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Setup the HomeKit accessory
	// ip := net.ParseIP(powerwallIP)
	bridgeInfo := accessory.Info{Name: "Tesla"}
	bridge := accessory.NewBridge(bridgeInfo)
	client := powerwall.NewClient(powerwallIP, username, password)
	interval, err := time.ParseDuration("10s")
	timeout, err := time.ParseDuration("60s")
	client.SetRetry(interval, timeout)
	result, err := client.GetStatus()
	if err != nil {
		panic(err)
	}
	fmt.Printf("The gateway's ID number is: %s\nIt is running version: %s\n", result.Din, result.Version)
	powerwall2 := powerwall_2.NewPowerwall(client)
	// gridSensor := grid.NewSensor(ip)

	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, bridge.A, powerwall2.A) //, powerwall.A, gridSensor.A)
	if err != nil {
		// stop if an error happens
		log.Panic(err)
	}

	// Setup a listener for interrupts and SIGTERM signals
	// to stop the server.
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		// Stop delivering signals.
		signal.Stop(c)
		// Cancel the context to stop the server.
		cancel()
	}()

	// Start the Prometheus exporter
	if prometheusExporter {
		go startPrometheus()
	}

	// Run the server.
	server.ListenAndServe(ctx)
}
