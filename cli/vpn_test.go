package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/config"
)

type fakeVPNInfoGetter struct {
	responses []*octaspace.SessionInfo
	calls     int
	called    chan struct{}
}

func (f *fakeVPNInfoGetter) Info(ctx context.Context) (*octaspace.SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.called != nil {
		select {
		case f.called <- struct{}{}:
		default:
		}
	}
	index := f.calls
	f.calls++
	if index >= len(f.responses) {
		return &octaspace.SessionInfo{}, nil
	}
	return f.responses[index], nil
}

type fakeVPNStopper struct {
	err             error
	calls           int
	contextCanceled bool
	hasDeadline     bool
}

func (f *fakeVPNStopper) Stop(ctx context.Context, _ *octaspace.StopParams) error {
	f.calls++
	f.contextCanceled = ctx.Err() != nil
	_, f.hasDeadline = ctx.Deadline()
	return f.err
}

func TestWaitForVPNConfig_ReturnsReadyConfig(t *testing.T) {
	getter := &fakeVPNInfoGetter{responses: []*octaspace.SessionInfo{
		{},
		{VPNConfig: "wg-config"},
	}}
	polls := 0
	got, err := waitForVPNConfig(context.Background(), getter, time.Second, time.Millisecond, func() {
		polls++
	})
	if err != nil {
		t.Fatalf("waitForVPNConfig: %v", err)
	}
	if got != "wg-config" {
		t.Fatalf("config = %q, want wg-config", got)
	}
	if getter.calls != 2 || polls != 1 {
		t.Fatalf("calls/polls = %d/%d, want 2/1", getter.calls, polls)
	}
}

func TestWaitForVPNConfig_StopsImmediatelyWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	getter := &fakeVPNInfoGetter{}

	_, err := waitForVPNConfig(ctx, getter, time.Second, 100*time.Millisecond, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if getter.calls != 0 {
		t.Fatalf("Info calls = %d, want 0", getter.calls)
	}
}

func TestWaitForVPNConfig_CancelsDuringPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	getter := &fakeVPNInfoGetter{called: make(chan struct{}, 1)}
	done := make(chan error, 1)
	go func() {
		_, err := waitForVPNConfig(ctx, getter, time.Second, time.Second, nil)
		done <- err
	}()

	select {
	case <-getter.called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waitForVPNConfig did not start polling")
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waitForVPNConfig did not stop after cancellation")
	}
}

func TestWaitForVPNConfig_ReturnsFriendlyTimeout(t *testing.T) {
	getter := &fakeVPNInfoGetter{}
	_, err := waitForVPNConfig(context.Background(), getter, 10*time.Millisecond, time.Millisecond, nil)
	if err == nil || !strings.Contains(err.Error(), "timed out waiting for VPN config") {
		t.Fatalf("error = %v, want timeout", err)
	}
}

func TestRollbackVPNConnect_UsesFreshBoundedContext(t *testing.T) {
	cause := errors.New("connect failed")
	stopper := &fakeVPNStopper{}
	err := rollbackVPNConnect(stopper, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want original cause", err)
	}
	if stopper.calls != 1 || stopper.contextCanceled || !stopper.hasDeadline {
		t.Fatalf("cleanup calls/canceled/deadline = %d/%v/%v", stopper.calls, stopper.contextCanceled, stopper.hasDeadline)
	}
}

func TestRollbackVPNConnect_JoinsCleanupError(t *testing.T) {
	cause := errors.New("connect failed")
	cleanupErr := errors.New("stop failed")
	stopper := &fakeVPNStopper{err: cleanupErr}
	err := rollbackVPNConnect(stopper, cause)
	if !errors.Is(err, cause) || !errors.Is(err, cleanupErr) {
		t.Fatalf("error = %v, want both causes", err)
	}
}

func TestActiveVPNSessionUUID(t *testing.T) {
	sessions := []octaspace.Session{
		{UUID: "mr-on-node", Service: "mr", NodeID: 7},
		{UUID: "vpn-on-node", Service: "vpn", NodeID: 7},
		{UUID: "saved-vpn", Service: "vpn", NodeID: 8},
	}

	if got := activeVPNSessionUUID(sessions, &config.Config{VPNRelayNode: 7}); got != "vpn-on-node" {
		t.Fatalf("node match = %q, want vpn-on-node", got)
	}
	if got := activeVPNSessionUUID(sessions, &config.Config{VPNRelayNode: 7, VPNSessionUUID: "saved-vpn"}); got != "saved-vpn" {
		t.Fatalf("saved match = %q, want saved-vpn", got)
	}
}

func TestShortSessionUUID(t *testing.T) {
	if got := shortSessionUUID("1234"); got != "1234" {
		t.Fatalf("short UUID = %q", got)
	}
	if got := shortSessionUUID("1234567890"); got != "12345678" {
		t.Fatalf("long UUID = %q", got)
	}
}
