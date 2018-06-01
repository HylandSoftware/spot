package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	colorable "github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
)

func initLogrus() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())
}

func watchAllTheThings(interval time.Duration, shutdown chan bool) {
	for {
		// do the thing
		log.Info("Running Watchdog Task")

		// wait for next interval
		select {
		case <-shutdown:
			return
		case <-time.After(interval):
			break
		}
	}
}

func main() {
	initLogrus()

	log.Info("Hello, World!")
	shutdown := make(chan bool)
	go watchAllTheThings(1*time.Second, shutdown)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	shutdown <- true
	log.Info("Goodbye")
}
