package main

import (
	"fmt"
	"github.com/UruhaLushia/sparkle-service/cmd"
	"os"
)

func main() {
	if err := cmd.MainCmd.Execute(); err != nil {
		if !cmd.IsReportedError(err) {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
