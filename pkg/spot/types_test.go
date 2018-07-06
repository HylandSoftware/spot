package spot

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDetector struct {
	mock.Mock
}

func (d *mockDetector) Name() string {
	args := d.Called()
	return fmt.Sprintf("[MockDetector] %s", args.String(0))
}

func (d *mockDetector) FindOfflineAgents() ([]string, error) {
	args := d.Called()
	return args.Get(0).([]string), args.Error(1)
}

type mockNotifier struct {
	mock.Mock
}

func setup(agents []string, e error) (*mockDetector, *mockNotifier, *Watchdog) {
	detector := &mockDetector{}
	notifier := &mockNotifier{}

	detector.On("Name").Return("a")
	detector.On("FindOfflineAgents").Return(agents, e)

	sut := NewWatchdog([]OfflineAgentDetector{detector}, notifier)

	return detector, notifier, sut
}

func (n *mockNotifier) Notify(agents map[string][]string) error {
	args := n.Called(agents)

	return args.Error(0)
}

func TestWatchdogRunChecksAndNotify_NoAgents(t *testing.T) {
	d, n, sut := setup([]string{}, nil)

	err := sut.RunChecksAndNotify()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertNotCalled(t, "Notify", map[string][]string{})
}

func TestWatchdogRunChecksAndNotify_Error(t *testing.T) {
	d, n, sut := setup(nil, fmt.Errorf("Mock Error"))

	err := sut.RunChecksAndNotify()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertNotCalled(t, "Notify", map[string][]string{})
}

func TestWatchdogRunChecksAndNotify_FoundAgents(t *testing.T) {
	offline := []string{"b", "c"}

	d, n, sut := setup(offline, nil)
	d.On("Name").Return("a")
	n.On("Notify", map[string][]string{"[MockDetector] a": {"b", "c"}}).Return(nil)

	err := sut.RunChecksAndNotify()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertCalled(t, "Notify", map[string][]string{"[MockDetector] a": {"b", "c"}})
}

func TestWatchdogRunChecks_DoesNotCallNotificationHandler(t *testing.T) {
	offline := []string{"b", "c"}

	d, n, sut := setup(offline, nil)
	d.On("Name").Return("a")

	result := sut.RunChecks()

	require.Equal(t, result, map[string][]string{"[MockDetector] a": {"b", "c"}})
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertNotCalled(t, "Notify", mock.AnythingOfType("map[string][]string"))
}

func TestWatchdogRunChecksAndNotify_NilNotificationHandler(t *testing.T) {
	offline := []string{"b", "c"}

	d, _, sut := setup(offline, nil)
	sut.NotificationHandler = nil

	err := sut.RunChecksAndNotify()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
}

func TestWatchdogRunChecksAndNotify_ConcatsAllOfflineForNotification(t *testing.T) {
	d, n, sut := setup([]string{"foo", "bar"}, nil)
	d.On("Name").Return("a")

	d2 := &mockDetector{}
	d2.On("Name").Return("d")
	d2.On("FindOfflineAgents").Return([]string{"fizz", "buzz"}, nil)

	sut.Detectors = append(sut.Detectors, d2)

	expected := map[string][]string{"[MockDetector] a": {"foo", "bar"}, "[MockDetector] d": {"fizz", "buzz"}}

	n.On("Notify", expected).Return(nil)

	err := sut.RunChecksAndNotify()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertCalled(t, "Notify", expected)
}

func TestCacheUpdate_NoSystems(t *testing.T) {
	sut := OfflineAgentCache{}

	result := sut.Update(map[string][]string{})

	require.Empty(t, result)
}

func TestCacheUpdate_MarksNewSystems(t *testing.T) {
	sut := OfflineAgentCache{}

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

func TestCacheUpdate_SilentForDuplicate(t *testing.T) {
	sut := OfflineAgentCache{}

	sut.Update(map[string][]string{"a": {"b"}})
	result := sut.Update(map[string][]string{"a": {"b"}})

	require.Empty(t, result)
}

func TestCacheUpdate_AddsToExistingSystem(t *testing.T) {
	sut := OfflineAgentCache{}

	sut.Update(map[string][]string{"a": {"b", "c"}})
	result := sut.Update(map[string][]string{"a": {"b", "c", "d"}})

	require.Contains(t, result, "a")
	require.Contains(t, result["a"], "d")
}

func TestCacheUpdate_RemovesNoLongerOfflineAgents(t *testing.T) {
	sut := OfflineAgentCache{}

	sut.Update(map[string][]string{"a": {"b", "c"}})
	result := sut.Update(map[string][]string{"a": {"c", "d"}})

	require.Contains(t, result, "a")
	require.NotContains(t, result["a"], "b")
	require.Contains(t, result["a"], "d")
}

func TestCacheUpdate_RemovesNoLongerOfflineSystems(t *testing.T) {
	sut := OfflineAgentCache{}

	sut.Update(map[string][]string{"a": {"b", "c"}, "e": {"f"}})
	result := sut.Update(map[string][]string{"a": {"c", "d"}})

	require.NotContains(t, result, "e")
	require.Contains(t, result, "a")
	require.NotContains(t, result["a"], "b")
	require.Contains(t, result["a"], "d")
}
