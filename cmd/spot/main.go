package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"bitbucket.hylandqa.net/do/spot/pkg/spot"
	colorable "github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
)

type dummyDetector struct{ id int }

func (d *dummyDetector) Name() string { return "[DummyDetector] http://dummy/" + strconv.Itoa(d.id) }
func (d *dummyDetector) FindOfflineAgents() ([]string, error) {
	if d.id == 2 {
		return nil, fmt.Errorf("something's FUBAR")
	}
	return []string{"a", "b", "c"}, nil
}

type dummyNotifier struct{}

func (d *dummyNotifier) Notify(agents []string) error { return nil }

func initLogrus() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())
}

func watchAllTheThings(interval time.Duration, w *spot.Watchdog, shutdown chan bool) {
	for {
		// do the thing
		if err := w.RunChecks(); err != nil {
			log.WithError(err).Error("Watchdog checks failed")
		}

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

	watchdog := &spot.Watchdog{
		Detectors: []spot.OfflineAgentDetector{
			&dummyDetector{id: 1},
			&dummyDetector{id: 2},
		},
		NotificationHandler: &dummyNotifier{},
	}

	shutdown := make(chan bool)
	go watchAllTheThings(1*time.Second, watchdog, shutdown)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	shutdown <- true
	log.Info("Goodbye")
}
