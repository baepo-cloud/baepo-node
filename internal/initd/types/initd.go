package types

import "os/exec"

type InitD interface {
	MainCmd() *exec.Cmd
}
