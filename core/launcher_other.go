//go:build !linux

package core

func newCoreLauncher() coreLauncher {
	return directCoreLauncher{}
}
