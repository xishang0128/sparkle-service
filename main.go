package main

import (
	"fmt"
	"os"
	"sparkle-service/cmd"
)

func main() {
	if err := cmd.MainCmd.Execute(); err != nil {
		if !cmd.IsReportedError(err) {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
