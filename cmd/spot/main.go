package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bitbucket.hylandqa.net/do/spot/pkg/spot"

	arg "github.com/alexflint/go-arg"
	colorable "github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
)

type applicationArgs struct {
	Jenkins   []string `arg:"-j,separate" help:"Jenkins Url & credentials in the form of https://jenkins/,username,password"`
	Verbosity string   `arg:"-v" help:"Verbosity [panic, fatal, error, warn, info, debug]"`
	Once      bool     `arg:"-o" help:"Run checks once and exit"`
}

func (applicationArgs) Description() string {
	return "alerts for disconnected build agents"
}

type dummyNotifier struct{}

func (d *dummyNotifier) Notify(agents []string) error { return nil }

func initLogrus(level string) {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())

	if level, err := log.ParseLevel(strings.ToLower(level)); err != nil {
		log.Panic(err)
	} else {
		log.SetLevel(level)
	}
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
	args := &applicationArgs{}
	args.Verbosity = "info"
	arg.MustParse(args)

	initLogrus(args.Verbosity)
	log.Info("Hello, World!")

	watchdog := &spot.Watchdog{
		Detectors:           []spot.OfflineAgentDetector{},
		NotificationHandler: &dummyNotifier{},
	}

	for _, v := range args.Jenkins {
		l := log.WithField("jenkins", v)
		l.Debug("Trying to parse jenkins instance")

		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			if len(parts) != 3 {
				l.Panicf("Expected url with credentials to have 3 parts but had %d", len(parts))
			}

			watchdog.Detectors = append(watchdog.Detectors, spot.NewJenkinsDetector(parts[0], parts[1], parts[2]))
		} else {
			watchdog.Detectors = append(watchdog.Detectors, spot.NewJenkinsDetector(v, "", ""))
		}
	}

	if len(watchdog.Detectors) == 0 {
		log.Panic("No detectors provided")
	}

	if args.Once {
		if err := watchdog.RunChecks(); err != nil {
			panic(err)
		}
	} else {
		shutdown := make(chan bool)
		go watchAllTheThings(10*time.Second, watchdog, shutdown)

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		shutdown <- true
	}

	log.Info("Goodbye")
}
