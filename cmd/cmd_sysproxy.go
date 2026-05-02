package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/UruhaLushia/sysproxy-go/sysproxy"
	"github.com/spf13/cobra"
)

var sysproxyCmd = &cobra.Command{
	Use:   "sysproxy",
	Short: "管理系统代理设置",
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "设置系统代理",
	Run: func(cmd *cobra.Command, args []string) {
		t := time.Now()
		err := sysproxy.SetProxy(&sysproxy.Options{
			Proxy:            server,
			Bypass:           bypass,
			Device:           device,
			OnlyActiveDevice: onlyActiveDevice,
			UseRegistry:      useRegistry,
		})
		if err != nil {
			fmt.Println("设置代理失败：", err)
			return
		}
		fmt.Println("代理设置成功，耗时：", time.Since(t))
	},
}

var pacCmd = &cobra.Command{
	Use:   "pac",
	Short: "设置 PAC 代理",
	Run: func(cmd *cobra.Command, args []string) {
		t := time.Now()
		err := sysproxy.SetPac(&sysproxy.Options{
			PACURL:           pacUrl,
			Device:           device,
			OnlyActiveDevice: onlyActiveDevice,
			UseRegistry:      useRegistry,
		})
		if err != nil {
			fmt.Println("设置 PAC 代理失败：", err)
			return
		}
		fmt.Println("PAC 代理设置成功，耗时：", time.Since(t))
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "取消代理设置",
	Run: func(cmd *cobra.Command, args []string) {
		t := time.Now()
		err := sysproxy.DisableProxy(&sysproxy.Options{
			Device:           device,
			OnlyActiveDevice: onlyActiveDevice,
			UseRegistry:      useRegistry,
		})
		if err != nil {
			fmt.Println("取消代理设置失败：", err)
			return
		}
		fmt.Println("代理设置已取消，耗时：", time.Since(t))
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看当前代理设置",
	Run: func(cmd *cobra.Command, args []string) {
		status, err := sysproxy.QueryProxySettings(&sysproxy.Options{
			Device:           device,
			OnlyActiveDevice: onlyActiveDevice,
			UseRegistry:      useRegistry,
		})
		if err != nil {
			fmt.Println("查询代理设置失败：", err)
			return
		}
		statusJSON, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			fmt.Println("格式化 JSON 失败：", err)
			return
		}
		fmt.Println(string(statusJSON))
	},
}
