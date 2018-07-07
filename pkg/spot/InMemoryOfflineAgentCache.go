package spot

type InMemoryOfflineAgentCache struct {
	backingCache map[string]map[string]bool
}

func NewInMemoryOfflineAgentCache() *InMemoryOfflineAgentCache {
	return &InMemoryOfflineAgentCache{
		backingCache: map[string]map[string]bool{},
	}
}

func (c InMemoryOfflineAgentCache) Update(offline map[string][]string) map[string][]string {
	result := map[string][]string{}

	for system, agents := range offline {
		// 1. Make entries for new systems
		if _, exists := c.backingCache[system]; !exists {
			c.backingCache[system] = map[string]bool{}
		}

		// 2. Make entries for new agents
		for _, agent := range agents {
			if _, exists := c.backingCache[system][agent]; !exists {
				c.backingCache[system][agent] = true
				result[system] = append(result[system], agent)
			}
		}

		// 3. Remove agents not in the offline list
		for agent := range c.backingCache[system] {
			found := false
			for _, a := range offline[system] {
				if agent == a {
					found = true
				}
			}

			if !found {
				delete(c.backingCache[system], agent)
			}
		}
	}

	// 4. Remove systems with no agents
	for system := range c.backingCache {
		if _, exists := offline[system]; !exists || len(c.backingCache[system]) == 0 {
			delete(c.backingCache, system)
		}
	}

	return result
}
