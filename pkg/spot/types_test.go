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
	d.Called()
	return "[MockDetector]"
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

	detector.On("Name").Once()
	detector.On("FindOfflineAgents").Return(agents, e)

	sut := &Watchdog{
		Detectors:           []OfflineAgentDetector{detector},
		NotificationHandler: notifier,
	}

	return detector, notifier, sut
}

func (n *mockNotifier) Notify(agents []string) error {
	args := n.Called(agents)

	if err := args.Error(0); err != nil {
		return err
	}

	return nil
}

func TestWatchdogRunChecks_NoAgents(t *testing.T) {
	d, n, sut := setup([]string{}, nil)

	err := sut.RunChecks()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertNotCalled(t, "Notify", []string{})
}

func TestWatchdogRunChecks_Error(t *testing.T) {
	d, n, sut := setup(nil, fmt.Errorf("Mock Error"))

	err := sut.RunChecks()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertNotCalled(t, "Notify", []string{})
}

func TestWatchdogRunChecks_FoundAgents(t *testing.T) {
	offline := []string{"foo", "bar"}

	d, n, sut := setup(offline, nil)
	n.On("Notify", offline).Return(nil)

	err := sut.RunChecks()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertCalled(t, "Notify", offline)
}

func TestWatchdogRunChecks_NilNotificationHandler(t *testing.T) {
	offline := []string{"foo", "bar"}

	d, _, sut := setup(offline, nil)
	sut.NotificationHandler = nil

	err := sut.RunChecks()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
}

func TestWatchdogRunChecks_ConcatsAllOfflineForNotification(t *testing.T) {
	d, n, sut := setup([]string{"foo", "bar"}, nil)

	d2 := &mockDetector{}
	d2.On("Name").Once()
	d2.On("FindOfflineAgents").Return([]string{"fizz", "buzz"}, nil)

	sut.Detectors = append(sut.Detectors, d2)

	n.On("Notify", []string{"foo", "bar", "fizz", "buzz"}).Return(nil)

	err := sut.RunChecks()

	require.Nil(t, err)
	d.AssertCalled(t, "FindOfflineAgents")
	n.AssertCalled(t, "Notify", []string{"foo", "bar", "fizz", "buzz"})
}
