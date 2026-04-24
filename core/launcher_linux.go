//go:build linux

package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

const disableLinuxSandboxEnv = "SPARKLE_CORE_DISABLE_LINUX_SANDBOX"

const linuxSandboxCloneFlags = syscall.CLONE_NEWNS |
	syscall.CLONE_NEWIPC |
	syscall.CLONE_NEWUTS

type linuxSandboxLauncher struct{}

type linuxSandboxMount struct {
	source   string
	target   string
	readOnly bool
	file     bool
	proc     bool
}

func newCoreLauncher() coreLauncher {
	if sandboxDisabled() {
		return directCoreLauncher{}
	}
	return linuxSandboxLauncher{}
}

func (linuxSandboxLauncher) Command(launch *launchSession) (*exec.Cmd, error) {
	root, cleanup, err := prepareLinuxSandboxRoot(launch)
	if err != nil {
		return nil, err
	}
	launch.addCleanup(func() {
		if err := cleanup(); err != nil {
			logSandboxCleanupError(err)
		}
	})

	cmd := exec.Command(launch.executablePath, launch.args...)
	cmd.Env = launch.env
	cmd.Dir = launch.workingDir
	configureCommand(cmd)

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Chroot = root
	cmd.SysProcAttr.Cloneflags |= linuxSandboxCloneFlags
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL

	return cmd, nil
}

func prepareLinuxSandboxRoot(launch *launchSession) (string, func() error, error) {
	root, err := os.MkdirTemp("", "sparkle-core-sandbox-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建核心沙盒目录失败：%w", err)
	}
	if err := prepareLinuxSandboxStaticLayout(root); err != nil {
		_ = os.RemoveAll(root)
		return "", nil, err
	}

	mounts, err := linuxSandboxMounts(launch)
	if err != nil {
		_ = os.RemoveAll(root)
		return "", nil, err
	}

	mounted := make([]string, 0, len(mounts))
	cleanup := func() error {
		var cleanupErr error
		for i := len(mounted) - 1; i >= 0; i-- {
			if err := syscall.Unmount(mounted[i], syscall.MNT_DETACH); err != nil && cleanupErr == nil {
				cleanupErr = fmt.Errorf("卸载沙盒映射失败 %s：%w", mounted[i], err)
			}
		}
		if err := os.RemoveAll(root); err != nil && cleanupErr == nil {
			cleanupErr = fmt.Errorf("清理核心沙盒目录失败：%w", err)
		}
		return cleanupErr
	}

	for _, mount := range mounts {
		target := sandboxTarget(root, mount.target)
		if err := mountIntoSandbox(target, mount); err != nil {
			_ = cleanup()
			return "", nil, err
		}
		mounted = append(mounted, target)
	}

	return root, cleanup, nil
}

func prepareLinuxSandboxStaticLayout(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "run"), 0o755); err != nil {
		return err
	}
	tmpDir := filepath.Join(root, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(tmpDir, 0o1777); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "var"), 0o755); err != nil {
		return err
	}
	if err := os.Symlink("/run", filepath.Join(root, "var", "run")); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func mountIntoSandbox(target string, mount linuxSandboxMount) error {
	if mount.proc {
		if err := os.MkdirAll(target, 0o555); err != nil {
			return err
		}
		if err := syscall.Mount("proc", target, "proc", uintptr(syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV), ""); err != nil {
			return fmt.Errorf("挂载 /proc 失败：%w", err)
		}
		return nil
	}

	if mount.file {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return err
		}
		_ = file.Close()
	} else if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}

	flags := uintptr(syscall.MS_BIND | syscall.MS_REC)
	if err := syscall.Mount(mount.source, target, "", flags, ""); err != nil {
		return fmt.Errorf("映射沙盒路径失败 %s -> %s：%w", mount.source, mount.target, err)
	}

	if mount.readOnly {
		flags |= syscall.MS_REMOUNT | syscall.MS_RDONLY
		if err := syscall.Mount(mount.source, target, "", flags, ""); err != nil {
			return fmt.Errorf("设置沙盒只读映射失败 %s：%w", mount.target, err)
		}
	}

	return nil
}

func linuxSandboxMounts(launch *launchSession) ([]linuxSandboxMount, error) {
	mounts := make([]linuxSandboxMount, 0, 32)
	addMount := func(source string, readOnly bool, file bool) error {
		source, err := normalizeSandboxPath(source)
		if err != nil {
			return err
		}
		if _, err := os.Stat(source); err != nil {
			return err
		}
		mounts = append(mounts, linuxSandboxMount{
			source:   source,
			target:   source,
			readOnly: readOnly,
			file:     file,
		})
		return nil
	}
	addMountIfExists := func(source string, readOnly bool, file bool) error {
		if _, err := os.Stat(source); os.IsNotExist(err) {
			return nil
		}
		return addMount(source, readOnly, file)
	}
	addDirForPath := func(path string, readOnly bool) error {
		dir, err := sandboxDirForPath(path)
		if err != nil {
			return err
		}
		return addMount(dir, readOnly, false)
	}
	addWritableDir := func(path string) error {
		path, err := normalizeSandboxPath(path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			path = filepath.Dir(path)
		}
		return addMount(path, false, false)
	}

	for _, dir := range []string{"/bin", "/sbin", "/usr", "/lib", "/lib64", "/etc", "/sys"} {
		if err := addMountIfExists(dir, true, false); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{"/run/systemd/resolve", "/run/resolvconf"} {
		if err := addMountIfExists(dir, true, false); err != nil {
			return nil, err
		}
	}

	if err := addDirForPath(launch.executablePath, true); err != nil {
		return nil, err
	}
	if executable, err := os.Executable(); err == nil {
		if err := addDirForPath(executable, true); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("读取 service 可执行文件路径失败：%w", err)
	}

	if err := addWritableDir(launch.workingDir); err != nil {
		return nil, err
	}
	if launch.logPath != "" {
		logDir := filepath.Dir(launch.logPath)
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return nil, err
		}
		if err := addWritableDir(logDir); err != nil {
			return nil, err
		}
	}
	if launch.hookUpFile != "" {
		hookDir := filepath.Dir(launch.hookUpFile)
		if err := addWritableDir(hookDir); err != nil {
			return nil, err
		}
	}

	for _, path := range launch.profile.SafePaths {
		if err := addWritableDir(path); err != nil {
			return nil, fmt.Errorf("映射可信路径失败 %q：%w", path, err)
		}
	}
	for _, path := range writableDirsFromCoreArgs(launch.args) {
		if err := addWritableDir(path); err != nil {
			return nil, err
		}
	}

	mounts = append(mounts, linuxSandboxMount{target: "/proc", proc: true})
	for _, dev := range []string{"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom", "/dev/net/tun"} {
		if err := addMountIfExists(dev, false, true); err != nil {
			return nil, err
		}
	}

	return compactSandboxMounts(mounts), nil
}

func compactSandboxMounts(mounts []linuxSandboxMount) []linuxSandboxMount {
	seen := make(map[string]int, len(mounts))
	result := make([]linuxSandboxMount, 0, len(mounts))
	for _, mount := range mounts {
		if mount.target == "" {
			continue
		}
		mount.target = filepath.Clean(mount.target)
		if mount.source != "" {
			mount.source = filepath.Clean(mount.source)
		}
		if index, ok := seen[mount.target]; ok {
			result[index] = mount
			continue
		}
		seen[mount.target] = len(result)
		result = append(result, mount)
	}

	slices.SortStableFunc(result, func(a, b linuxSandboxMount) int {
		return strings.Count(a.target, string(os.PathSeparator)) - strings.Count(b.target, string(os.PathSeparator))
	})
	return result
}

func writableDirsFromCoreArgs(args []string) []string {
	var dirs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		value := ""
		switch arg {
		case "-d", "-ext-ctl-unix":
			if i+1 < len(args) {
				value = args[i+1]
				i++
			}
		default:
			if after, ok := strings.CutPrefix(arg, "-d="); ok {
				value = after
			} else if after, ok := strings.CutPrefix(arg, "-ext-ctl-unix="); ok {
				value = after
			}
		}

		if value == "" {
			continue
		}
		if arg == "-ext-ctl-unix" || strings.HasPrefix(arg, "-ext-ctl-unix=") {
			value = filepath.Dir(value)
		}
		dirs = append(dirs, value)
	}
	return dirs
}

func sandboxDirForPath(path string) (string, error) {
	path, err := normalizeSandboxPath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}

func sandboxTarget(root string, target string) string {
	return filepath.Join(root, strings.TrimPrefix(filepath.Clean(target), string(os.PathSeparator)))
}

func normalizeSandboxPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("路径不能为空")
	}
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = absPath
	}
	return filepath.Clean(path), nil
}

func sandboxDisabled() bool {
	value := strings.TrimSpace(os.Getenv(disableLinuxSandboxEnv))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func logSandboxCleanupError(err error) {
	if err != nil {
		log.Printf("清理核心沙盒失败：%v", err)
	}
}
