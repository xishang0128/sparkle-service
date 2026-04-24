package cmd

import (
	"os"
	"path/filepath"
	"sparkle-service/log"
	"sparkle-service/route"
	appservice "sparkle-service/service"

	kservice "github.com/kardianos/service"
	"github.com/spf13/cobra"
)

type Program struct {
	listen string
}

func (p *Program) Start(s kservice.Service) error {
	go p.run()
	return nil
}

func (p *Program) run() {
	logFile, err := log.InitLogging()
	if err != nil {
		log.Printf("初始化日志失败：%v\n", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	log.Println("服务启动中...")

	if err := route.Start(p.listen); err != nil {
		log.Fatal(err)
	}
}

func (p *Program) Stop(s kservice.Service) error {
	log.Println("服务停止中...")
	if err := route.Stop(); err != nil {
		log.Printf("服务停止清理失败：%v", err)
		return err
	}
	log.Println("服务已停止")
	return nil
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Sparkle 服务",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}

		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, os.Args[0])
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		if err := s.Install(); err != nil {
			log.Println("安装服务失败：", err)
			return
		}
		if err := s.Start(); err != nil {
			log.Println("启动服务失败：", err)
			return
		}
		log.Println("服务安装成功")
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "卸载 Sparkle 服务",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		if err := s.Stop(); err != nil {
			log.Println("停止服务失败：", err)
			return
		}
		if err := s.Uninstall(); err != nil {
			log.Println("卸载服务失败：", err)
			return
		}
		log.Println("服务卸载成功")
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Sparkle 服务",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		if err := s.Start(); err != nil {
			log.Println("启动服务失败：", err)
			return
		}
		log.Println("服务启动成功")
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Sparkle 服务",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		if err := s.Stop(); err != nil {
			log.Println("停止服务失败：", err)
			return
		}
		log.Println("服务停止成功")
	},
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 Sparkle 服务",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		if err := s.Restart(); err != nil {
			log.Println("重启服务失败：", err)
			return
		}
		log.Println("服务重启成功")
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 Sparkle 服务状态",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		status, err := s.Status()
		if err != nil {
			log.Println("查询服务状态失败：", err)
			return
		}

		switch status {
		case kservice.StatusRunning:
			log.Println("服务状态：运行中")
		case kservice.StatusStopped:
			log.Println("服务状态：已停止")
		case kservice.StatusUnknown:
			log.Println("服务状态：未知")
		default:
			log.Println("服务状态：未知")
		}
	},
}

var serviceRunCmd = &cobra.Command{
	Use:   "run",
	Short: "运行 Sparkle 服务",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureServiceRuntimeExecutable(); err != nil {
			log.Fatal(err)
		}

		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Fatal(err)
		}

		if err := s.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "管理 Sparkle 服务",
}

var serviceInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化服务（传入公钥）",
	Run: func(cmd *cobra.Command, args []string) {
		publicKey := cmd.Flag("public-key").Value.String()
		authorizedSID := cmd.Flag("authorized-sid").Value.String()
		authorizedUID, _ := cmd.Flags().GetUint32("authorized-uid")
		if publicKey == "" {
			log.Println("错误：必须通过 --public-key 参数提供公钥")
			return
		}
		if authorizedSID == "" && !cmd.Flags().Changed("authorized-uid") {
			log.Println("错误：必须通过 --authorized-sid 或 --authorized-uid 绑定允许访问服务的用户身份")
			return
		}
		userDataDir := route.GetConfigDir()
		keyDir := filepath.Join(userDataDir, "sparkle", "keys")

		_ = route.InitKeyManager(keyDir)

		km := route.GetKeyManager()
		keyChanged, err := km.SetPublicKey(publicKey)
		if err != nil {
			log.Println("设置公钥失败：", err)
			return
		}

		principalChanged := false
		switch {
		case authorizedSID != "":
			principalChanged, err = km.SetAuthorizedSID(authorizedSID)
			if err != nil {
				log.Println("设置授权 SID 失败：", err)
				return
			}
		case cmd.Flags().Changed("authorized-uid"):
			principalChanged, err = km.SetAuthorizedUID(authorizedUID)
			if err != nil {
				log.Println("设置授权 UID 失败：", err)
				return
			}
		}

		if keyChanged || principalChanged {
			log.Println("服务初始化成功，认证配置已更新")
		} else {
			log.Println("服务初始化成功，认证配置未变化")
		}

		listenAddr := listen
		if listenAddr == "" {
			listenAddr = defaultAddr
		}
		prg := &Program{listen: listenAddr}
		s, err := appservice.New(prg, "")
		if err != nil {
			log.Println("创建服务失败：", err)
			return
		}

		status, err := s.Status()
		if err != nil {
			log.Println("查询服务状态失败：", err)
			log.Println("提示：如果服务正在运行，请手动执行 'restart' 命令")
			return
		}

		if status == kservice.StatusRunning {
			if !keyChanged && !principalChanged {
				log.Println("服务已在运行，配置未变化，无需重启")
				return
			}
			log.Println("正在重启服务...")
			if err := s.Restart(); err != nil {
				log.Println("重启服务失败：", err)
				log.Println("请手动执行 'sparkle-service service restart' 命令")
				return
			}
			log.Println("服务已成功重启")
		} else {
			log.Println("服务未运行，配置将在下次启动时生效")
		}
	},
}

func init() {
	serviceCmd.AddCommand(serviceInitCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceRunCmd)

	serviceInitCmd.Flags().StringP("public-key", "k", "", "客户端公钥")
	serviceInitCmd.Flags().String("authorized-sid", "", "允许访问服务的 Windows SID")
	serviceInitCmd.Flags().Uint32("authorized-uid", 0, "允许访问服务的 Unix UID")
}
