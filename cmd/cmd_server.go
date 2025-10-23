package cmd

import (
	"sparkle-service/log"
	"sparkle-service/route"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 Sparkle 服务（测试用）",
	Run: func(cmd *cobra.Command, args []string) {
		if err := route.StartHTTP("127.0.0.1:10002"); err != nil {
			log.Fatal(err)
		}
	},
}
