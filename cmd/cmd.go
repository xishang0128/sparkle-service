package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
)

var (
	server string
	bypass string
	pacUrl string

	device           string
	onlyActiveDevice bool

	listen      string
	defaultAddr string
)

var MainCmd = &cobra.Command{
	Use:   "sparkle-service",
	Short: "Sparkle Service",
}

func init() {
	if runtime.GOOS == "windows" {
		defaultAddr = "\\\\.\\pipe\\sparkle\\service"
	} else {
		defaultAddr = "/tmp/sparkle-service.sock"
	}

	MainCmd.AddCommand(proxyCmd)
	MainCmd.AddCommand(pacCmd)
	MainCmd.AddCommand(disableCmd)
	MainCmd.AddCommand(statusCmd)
	MainCmd.AddCommand(serverCmd)
	MainCmd.AddCommand(serviceCmd)

	MainCmd.PersistentFlags().BoolVarP(&onlyActiveDevice, "only-active-device", "a", false, "仅对活跃的网络设备生效")
	MainCmd.PersistentFlags().StringVarP(&device, "device", "d", "", "指定网络设备")
	MainCmd.PersistentFlags().StringVarP(&listen, "listen", "l", defaultAddr, "监听地址")

	proxyCmd.Flags().StringVarP(&server, "server", "s", "", "代理服务器地址")
	proxyCmd.Flags().StringVarP(&bypass, "bypass", "b", "", "绕过地址")

	pacCmd.Flags().StringVarP(&pacUrl, "url", "u", "", "pac 地址")
}
