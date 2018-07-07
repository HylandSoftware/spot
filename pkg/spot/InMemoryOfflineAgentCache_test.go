package spot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdate_NoSystems(t *testing.T) {
	sut := NewInMemoryOfflineAgentCache()

	result := sut.Update(map[string][]string{})

	require.Empty(t, result)
}

func TestUpdate_MarksNewSystems(t *testing.T) {
	sut := NewInMemoryOfflineAgentCache()

	result := sut.Update(map[string][]string{
		"a": {"b", "c"},
		"d": {"e", "f"},
	})

	require.Contains(t, result, "a")
	require.Contains(t, result["a"], "b")
	require.Contains(t, result["a"], "c")

	require.Contains(t, result, "d")
	require.Contains(t, result["d"], "e")
	require.Contains(t, result["d"], "f")
}

func TestUpdate_SilentForDuplicate(t *testing.T) {
	sut := NewInMemoryOfflineAgentCache()

	sut.Update(map[string][]string{"a": {"b"}})
	result := sut.Update(map[string][]string{"a": {"b"}})

	require.Empty(t, result)
}

func TestUpdate_AddsToExistingSystem(t *testing.T) {
	sut := NewInMemoryOfflineAgentCache()

	sut.Update(map[string][]string{"a": {"b", "c"}})
	result := sut.Update(map[string][]string{"a": {"b", "c", "d"}})

	require.Contains(t, result, "a")
	require.Contains(t, result["a"], "d")
}

func TestUpdate_RemovesNoLongerOfflineAgents(t *testing.T) {
	sut := NewInMemoryOfflineAgentCache()

	sut.Update(map[string][]string{"a": {"b", "c"}})
	result := sut.Update(map[string][]string{"a": {"c", "d"}})

	require.Contains(t, result, "a")
	require.NotContains(t, result["a"], "b")
	require.Contains(t, result["a"], "d")
}

func TestUpdate_RemovesNoLongerOfflineSystems(t *testing.T) {
	sut := NewInMemoryOfflineAgentCache()

	sut.Update(map[string][]string{"a": {"b", "c"}, "e": {"f"}})
	result := sut.Update(map[string][]string{"a": {"c", "d"}})

	require.NotContains(t, result, "e")
	require.Contains(t, result, "a")
	require.NotContains(t, result["a"], "b")
	require.Contains(t, result["a"], "d")
}
