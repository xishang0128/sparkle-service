//go:build linux

package core

import (
	"errors"
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

const linuxSandboxCloneFlags = syscall.CLONE_NEWIPC |
	syscall.CLONE_NEWUTS

const linuxSandboxUnshareFlags = syscall.CLONE_NEWNS

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
	cmd.SysProcAttr.Unshareflags |= linuxSandboxUnshareFlags
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL

	return cmd, nil
}

func prepareLinuxSandboxRoot(launch *launchSession) (string, func() error, error) {
	cleanupStaleLinuxSandboxRoots()

	root, err := os.MkdirTemp("", "sparkle-core-sandbox-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建核心沙盒目录失败：%w", err)
	}
	if err := mountSandboxRoot(root); err != nil {
		_ = os.RemoveAll(root)
		return "", nil, err
	}
	if err := prepareLinuxSandboxStaticLayout(root); err != nil {
		_ = cleanupLinuxSandboxRoot(root)
		return "", nil, err
	}

	mounts, err := linuxSandboxMounts(launch)
	if err != nil {
		_ = cleanupLinuxSandboxRoot(root)
		return "", nil, err
	}

	mounted := make([]string, 0, len(mounts))
	cleanup := func() error {
		var cleanupErr error
		for i := len(mounted) - 1; i >= 0; i-- {
			if err := makeSandboxMountPrivate(mounted[i]); err != nil {
				if cleanupErr == nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOENT) {
					cleanupErr = fmt.Errorf("隔离沙盒映射失败 %s：%w", mounted[i], err)
				}
				continue
			}
			if err := syscall.Unmount(mounted[i], syscall.MNT_DETACH); err != nil && cleanupErr == nil {
				cleanupErr = fmt.Errorf("卸载沙盒映射失败 %s：%w", mounted[i], err)
			}
		}
		if err := makeSandboxMountPrivate(root); err != nil {
			if cleanupErr == nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOENT) {
				cleanupErr = fmt.Errorf("隔离核心沙盒根目录失败 %s：%w", root, err)
			}
		} else if err := syscall.Unmount(root, syscall.MNT_DETACH); err != nil &&
			cleanupErr == nil &&
			!errors.Is(err, syscall.EINVAL) &&
			!errors.Is(err, syscall.ENOENT) {
			cleanupErr = fmt.Errorf("卸载核心沙盒根目录失败 %s：%w", root, err)
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

func mountSandboxRoot(root string) error {
	if err := syscall.Mount(root, root, "", uintptr(syscall.MS_BIND), ""); err != nil {
		return fmt.Errorf("初始化核心沙盒根目录失败 %s：%w", root, err)
	}
	if err := makeSandboxMountPrivate(root); err != nil {
		_ = syscall.Unmount(root, syscall.MNT_DETACH)
		return fmt.Errorf("隔离核心沙盒根目录失败 %s：%w", root, err)
	}
	return nil
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

func cleanupStaleLinuxSandboxRoots() {
	roots, err := filepath.Glob(filepath.Join(os.TempDir(), "sparkle-core-sandbox-*"))
	if err != nil {
		logSandboxCleanupError(fmt.Errorf("查找残留核心沙盒目录失败：%w", err))
		return
	}

	for _, root := range roots {
		if err := cleanupLinuxSandboxRoot(root); err != nil {
			logSandboxCleanupError(err)
		}
	}
}

func cleanupLinuxSandboxRoot(root string) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取残留核心沙盒目录失败 %s：%w", root, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	var cleanupErr error
	for _, mountPoint := range linuxMountPointsUnder(root) {
		if err := makeSandboxMountPrivate(mountPoint); err != nil {
			if !errors.Is(err, syscall.EINVAL) &&
				!errors.Is(err, syscall.ENOENT) &&
				cleanupErr == nil {
				cleanupErr = fmt.Errorf("隔离残留核心沙盒映射失败 %s：%w", mountPoint, err)
			}
			continue
		}
		if err := syscall.Unmount(mountPoint, syscall.MNT_DETACH); err != nil &&
			!errors.Is(err, syscall.EINVAL) &&
			!errors.Is(err, syscall.ENOENT) &&
			cleanupErr == nil {
			cleanupErr = fmt.Errorf("卸载残留核心沙盒映射失败 %s：%w", mountPoint, err)
		}
	}
	if err := os.RemoveAll(root); err != nil && cleanupErr == nil {
		cleanupErr = fmt.Errorf("清理残留核心沙盒目录失败 %s：%w", root, err)
	}
	return cleanupErr
}

func linuxMountPointsUnder(root string) []string {
	root = filepath.Clean(root)
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil
	}

	var mountPoints []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		mountPoint := filepath.Clean(unescapeMountInfoPath(fields[4]))
		if pathWithin(mountPoint, root) {
			mountPoints = append(mountPoints, mountPoint)
		}
	}

	slices.SortFunc(mountPoints, func(a, b string) int {
		return strings.Count(b, string(os.PathSeparator)) - strings.Count(a, string(os.PathSeparator))
	})
	return mountPoints
}

func unescapeMountInfoPath(path string) string {
	replacer := strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, `\`,
	)
	return replacer.Replace(path)
}

func pathWithin(path string, root string) bool {
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func mountIntoSandbox(target string, mount linuxSandboxMount) error {
	if mount.proc {
		return mountKernelFilesystem(target, "proc", uintptr(syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV))
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
	if err := makeSandboxMountPrivate(target); err != nil {
		_ = syscall.Unmount(target, syscall.MNT_DETACH)
		return fmt.Errorf("隔离沙盒映射失败 %s：%w", mount.target, err)
	}

	if mount.readOnly {
		flags |= syscall.MS_REMOUNT | syscall.MS_RDONLY
		if err := syscall.Mount(mount.source, target, "", flags, ""); err != nil {
			_ = syscall.Unmount(target, syscall.MNT_DETACH)
			return fmt.Errorf("设置沙盒只读映射失败 %s：%w", mount.target, err)
		}
	}

	return nil
}

func makeSandboxMountPrivate(target string) error {
	err := syscall.Mount("", target, "", uintptr(syscall.MS_PRIVATE|syscall.MS_REC), "")
	if errors.Is(err, syscall.EINVAL) {
		return syscall.Mount("", target, "", uintptr(syscall.MS_PRIVATE), "")
	}
	return err
}

func mountKernelFilesystem(target string, fsType string, flags uintptr) error {
	if err := os.MkdirAll(target, 0o555); err != nil {
		return err
	}
	if err := syscall.Mount(fsType, target, fsType, flags, ""); err != nil {
		return fmt.Errorf("挂载 %s 失败：%w", target, err)
	}
	if err := makeSandboxMountPrivate(target); err != nil {
		_ = syscall.Unmount(target, syscall.MNT_DETACH)
		return fmt.Errorf("隔离沙盒映射失败 %s：%w", target, err)
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
		path, err := writableSandboxDir(path)
		if err != nil {
			return err
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

	coreDir, err := sandboxDirForPath(launch.executablePath)
	if err != nil {
		return nil, err
	}
	if err := addMount(coreDir, false, false); err != nil {
		return nil, err
	}
	if executable, err := os.Executable(); err == nil {
		if err := addDirForPath(executable, true); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("读取 service 可执行文件路径失败：%w", err)
	}

	workingDir, err := sandboxDirForPath(launch.workingDir)
	if err != nil {
		return nil, err
	}
	if workingDir != coreDir {
		if err := addMount(workingDir, true, false); err != nil {
			return nil, err
		}
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

func writableSandboxDir(path string) (string, error) {
	path, err := normalizeSandboxPath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	path = filepath.Clean(path)

	return path, nil
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
