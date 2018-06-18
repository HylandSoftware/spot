package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bitbucket.hylandqa.net/do/spot/pkg/spot"
	"bitbucket.hylandqa.net/do/spot/pkg/spot/bamboo"
	"bitbucket.hylandqa.net/do/spot/pkg/spot/jenkins"

	arg "github.com/alexflint/go-arg"
	colorable "github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
)

type applicationArgs struct {
	Bamboo    []string `arg:"-b,separate" help:"Bamboo Url & credentials in the form of https://bamboo/,username,password"`
	Jenkins   []string `arg:"-j,separate" help:"Jenkins Url & credentials in the form of https://jenkins/,username,password"`
	Slack     string   `arg:"-s" help:"Slack-Compatible Incoming Webhook URL"`
	Template  string   `arg:"-t" help:"Path to template for notifications"`
	Verbosity string   `arg:"-v" help:"Verbosity [panic, fatal, error, warn, info, debug]"`
	Period    string   `arg:"-p" help:"How long to wait between checks"`
	Once      bool     `arg:"-o" help:"Run checks once and exit"`
	WarmUp    bool     `arg:"-w" help:"Run checks without notifications once before starting the watchdog"`
}

func (applicationArgs) Description() string {
	return "alerts for disconnected build agents"
}

type dummyNotifier struct{}

func (d *dummyNotifier) Notify(agents map[string][]string) error { return nil }

func initLogrus(level string) {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())

	if level, err := log.ParseLevel(strings.ToLower(level)); err != nil {
		log.Panic(err)
	} else {
		log.SetLevel(level)
	}
}

func waitOne(sig <-chan bool, delay <-chan time.Time) bool {
	select {
	case <-sig:
		return false
	case <-delay:
		return true
	}
}

func watchAllTheThings(warmUp bool, interval time.Duration, w *spot.Watchdog, shutdown <-chan bool) {
	done := make(chan time.Time)

	if warmUp {
		log.Info("Warming the offline cache")
		go func() {
			w.RunChecks()
			done <- time.Now()
		}()

		if !waitOne(shutdown, done) {
			return
		}

		if !waitOne(shutdown, time.After(interval)) {
			return
		}
	}

	for {
		// do the thing
		go func() {
			if err := w.RunChecksAndNotify(); err != nil {
				log.WithError(err).Error("Watchdog checks failed")
			}
			done <- time.Now()
		}()

		if !waitOne(shutdown, done) {
			return
		}

		// wait for next interval
		if !waitOne(shutdown, time.After(interval)) {
			return
		}
	}
}

func (a *applicationArgs) populateBamboo(p *arg.Parser) []spot.OfflineAgentDetector {
	result := []spot.OfflineAgentDetector{}

	for _, v := range a.Bamboo {
		l := log.WithField("bamboo", v)
		l.Debug("Trying to parse bamboo instance")

		if detector, err := bamboo.NewBambooDetectorFromArg(v); err != nil {
			p.Fail(fmt.Sprintf("Failed to parse bamboo configuration: %s", err.Error()))
		} else {
			result = append(result, spot.OfflineAgentDetector(detector))
		}
	}

	return result
}

func (a *applicationArgs) populateJenkins(p *arg.Parser) []spot.OfflineAgentDetector {
	result := []spot.OfflineAgentDetector{}

	for _, v := range a.Jenkins {
		l := log.WithField("jenkins", v)
		l.Debug("Trying to parse jenkins instance")

		if detector, err := jenkins.NewJenkinsDetectorFromArg(v); err != nil {
			p.Fail(fmt.Sprintf("Failed to parse jenkins configuration: %s", err.Error()))
		} else {
			result = append(result, spot.OfflineAgentDetector(detector))
		}
	}

	return result
}

func main() {
	args := &applicationArgs{}
	args.Verbosity = "info"
	p := arg.MustParse(args)

	initLogrus(args.Verbosity)
	log.Info("Hello, World!")

	detectors := []spot.OfflineAgentDetector{}
	var handler spot.Notifier = &dummyNotifier{}

	if args.Slack != "" {
		var err error
		if handler, err = spot.NewSlackNotifier(args.Slack, args.Template); err != nil {
			p.Fail(fmt.Sprintf("Invalid slack URL: %s", err.Error()))
		}
	}

	bambooDetectors := args.populateBamboo(p)
	detectors = append(detectors, bambooDetectors...)

	jenkinsDetectors := args.populateJenkins(p)
	detectors = append(detectors, jenkinsDetectors...)

	if len(detectors) == 0 {
		p.Fail("Provide at least one watchdog configuration")
	}

	watchdog := spot.NewWatchdog(detectors, handler)

	if args.Once {
		if err := watchdog.RunChecksAndNotify(); err != nil {
			panic(err)
		}
	} else {
		period, err := time.ParseDuration(args.Period)
		if err != nil {
			p.Fail(fmt.Sprintf("Failed to parse period: %s", err.Error()))
		}

		shutdown := make(chan bool)
		go watchAllTheThings(args.WarmUp, period, watchdog, shutdown)

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		shutdown <- true
	}

	log.Info("Goodbye")
}
