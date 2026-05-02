//go:build windows

package core

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/UruhaLushia/sparkle-service/listen"
)

func TestCmdEchoNamedPipe(t *testing.T) {
	token := "test-token"
	pipePath := `\\.\pipe\sparkle\core-ready-test-` + fmt.Sprint(time.Now().UnixNano())
	listener, err := listen.ListenNamedPipe(pipePath, "")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	read := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			read <- "accept: " + err.Error()
			return
		}
		defer conn.Close()
		data, err := io.ReadAll(io.LimitReader(conn, 128))
		if err != nil {
			read <- "read: " + err.Error()
			return
		}
		read <- strings.TrimSpace(string(data))
	}()

	cmd := exec.Command("cmd.exe", "/C", "echo "+token+" > "+pipePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, output)
	}

	select {
	case got := <-read:
		if got != token {
			t.Fatalf("got %q, want %q", got, token)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
