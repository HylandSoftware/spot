package spot

import (
	log "github.com/sirupsen/logrus"
)

// Watchdog holds a reference to a set of detectors and a Notification handler
type Watchdog struct {
	Detectors           []OfflineAgentDetector
	NotificationHandler Notifier

	cache OfflineAgentCache
}

// NewWatchdog constructs a Watchdog
func NewWatchdog(detectors []OfflineAgentDetector, handler Notifier) *Watchdog {
	return &Watchdog{
		Detectors:           detectors,
		NotificationHandler: handler,

		cache: OfflineAgentCache{},
	}
}

// RunChecks polls all detectors. If any detector returns one or more offline
// agent, they are added to the map that is returned
func (w *Watchdog) RunChecks() map[string][]string {
	found := map[string][]string{}

	log.Info("Running Watchdog Task")
	for _, v := range w.Detectors {
		l := log.WithField("detector", v.Name())

		l.Debug("Checking for offline agents")

		if offline, err := v.FindOfflineAgents(); err != nil {
			l.WithError(err).Error("Failed to check for offline agents")
		} else if len(offline) > 0 {
			l.WithField("offline", offline).Warn("One or more agents are offline")
			found[v.Name()] = offline
		}

		l.Debug("Check Complete")
	}

	return w.cache.Update(found)
}

// RunChecksAndNotify calls w.RunChecks. If Any offline agents are returned
// a notification is sent
func (w *Watchdog) RunChecksAndNotify() error {
	toNotify := w.RunChecks()

	if len(toNotify) > 0 {
		if w.NotificationHandler == nil {
			log.Error("No notification handler")
			return nil
		}

		log.Info("Sending Notification")
		return w.NotificationHandler.Notify(toNotify)
	}

	log.Info("No newly offline agents")
	return nil
}

// OfflineAgentDetector is the basic unit-of-work for spot. Each detector
// knows how to talk to a build system and determine which agents are offline.
type OfflineAgentDetector interface {
	// Name returns the name of the detector. It may include more detailed
	// information if more than one detector of the same type is expected
	// to be used.
	Name() string

	/// FindOfflineAgents returns a string array of agents that are offline.
	FindOfflineAgents() ([]string, error)
}

// OfflineAgentCache remembers what agents are still offline
type OfflineAgentCache map[string]map[string]bool

// Update updates the cache and returns a map of systems to agents that are newly offline
func (c OfflineAgentCache) Update(offline map[string][]string) map[string][]string {
	result := map[string][]string{}

	for system, agents := range offline {
		// 1. Make entries for new systems
		if _, exists := c[system]; !exists {
			c[system] = map[string]bool{}
		}

		// 2. Make entries for new agents
		for _, agent := range agents {
			if _, exists := c[system][agent]; !exists {
				c[system][agent] = true
				result[system] = append(result[system], agent)
			}
		}

		// 3. Remove agents not in the offline list
		for agent := range c[system] {
			found := false
			for _, a := range offline[system] {
				if agent == a {
					found = true
				}
			}

			if !found {
				delete(c[system], agent)
			}
		}
	}

	// 4. Remove systems with no agents
	for system := range c {
		if _, exists := offline[system]; !exists || len(c[system]) == 0 {
			delete(c, system)
		}
	}

	return result
}

// Notifier provides a way to warn interested parties about offline agents.
type Notifier interface {
	// Notify takes an map of detector names to array of offline agents and
	// sends a notification, optionally returning an error.
	Notify(agents map[string][]string) error
}
