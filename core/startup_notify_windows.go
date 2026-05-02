//go:build windows

package core

import (
	"fmt"

	"github.com/UruhaLushia/sparkle-service/core/security"
	"github.com/UruhaLushia/sparkle-service/listen"
)

func createNativeStartupHook(token string) (*coreStartupHook, error) {
	pipePath := `\\.\pipe\sparkle\core-ready-` + token
	listener, err := listen.ListenNamedPipe(pipePath, currentProcessPipeSDDL())
	if err != nil {
		return nil, fmt.Errorf("创建核心启动通知管道失败：%w", err)
	}

	return newCoreStartupHook(listener, token, pipePath, "echo "+token+" > "+pipePath, noopShellCommand(), nil), nil
}

func currentProcessPipeSDDL() string {
	sid, err := security.CurrentProcessSID()
	if err != nil {
		return "D:P(A;;GA;;;SY)(A;;GA;;;BA)"
	}
	return fmt.Sprintf("D:P(A;;GA;;;%s)(A;;GA;;;SY)(A;;GA;;;BA)", sid.String())
}
