package route

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sparkle-service/log"
	"syscall"

	"sparkle-service/listen"
)

var (
	unixServer *http.Server
	pipeServer *http.Server
)

func GetConfigDir() string {
	if dir := os.Getenv("SPARKLE_CONFIG_DIR"); dir != "" {
		return dir
	}

	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.ExpandEnv("$HOME")
	}

	if homeDir == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			return dir
		}
	}

	switch runtime.GOOS {
	case "darwin":
		if homeDir == "" || os.Getuid() == 0 {
			homeDir = "/var/root"
		}
		return filepath.Join(homeDir, "Library", "Application Support")
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return appdata
		}
		if dir, err := os.UserConfigDir(); err == nil {
			return dir
		}
	default:
		return filepath.Join(homeDir, ".config")
	}

	return filepath.Join(homeDir, ".config")
}

func Start(addr string) error {
	userDataDir := GetConfigDir()

	keyDir := filepath.Join(userDataDir, "sparkle", "keys")

	log.Printf("配置目录: %s", userDataDir)
	log.Printf("密钥目录: %s", keyDir)

	if err := InitKeyManager(keyDir); err != nil {
		log.Printf("警告: 初始化密钥管理器失败: %v", err)
	}

	km := GetKeyManager()
	if km.IsInitialized() {
		log.Println("密钥管理器已初始化")
	} else {
		log.Println("警告：密钥管理器未初始化")
	}

	if runtime.GOOS == "windows" {
		if err := startServer(addr, StartPipe); err != nil {
			return err
		}
	} else {
		if err := startServer(addr, StartUnix); err != nil {
			return err
		}
	}
	if runtime.GOOS == "windows" {
		if err := startServer("127.0.0.1:10001", StartHTTP); err != nil {
			return err
		}
	} else {
		if err := startServer("127.0.0.1:10010", StartHTTP); err != nil {
			return err
		}
	}

	return nil
}

func startServer(addr string, startFunc func(string) error) error {
	if unixServer != nil {
		_ = unixServer.Close()
		unixServer = nil
	}

	if pipeServer != nil {
		_ = pipeServer.Close()
		pipeServer = nil
	}

	if len(addr) > 0 {
		dir := filepath.Dir(addr)
		if err := ensureDirExists(dir); err != nil {
			return err
		}

		if err := syscall.Unlink(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unlink 错误：%w", err)
		}

		if err := startFunc(addr); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("目录创建错误：%w", err)
		}
	}
	return nil
}

func StartHTTP(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http 监听错误：%w", err)
	}
	log.Printf("http 监听地址: %s", addr)
	server := &http.Server{
		Handler: router(),
	}
	return server.Serve(l)
}

func StartUnix(addr string) error {
	l, err := net.Listen("unix", addr)
	if err != nil {
		return fmt.Errorf("unix 监听错误：%w", err)
	}
	_ = os.Chmod(addr, 0o666)
	log.Printf("unix 监听地址: %s", l.Addr().String())

	server := &http.Server{
		Handler: router(),
	}
	unixServer = server
	return server.Serve(l)
}

func StartPipe(addr string) error {
	l, err := listen.ListenNamedPipe(addr)
	if err != nil {
		return fmt.Errorf("pipe 监听错误：%w", err)
	}
	log.Printf("pipe 监听地址: %s", l.Addr().String())

	server := &http.Server{
		Handler: router(),
	}
	pipeServer = server
	return server.Serve(l)
}
