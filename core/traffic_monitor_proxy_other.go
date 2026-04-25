//go:build !windows

package core

func startTrafficMonitorProxy(_ *launchSession, _ string) (func(), error) {
	return nil, nil
}
