package main

import (
	"fmt"
	"os"
	"sparkle-service/cmd"
)

func main() {
	if err := cmd.MainCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
