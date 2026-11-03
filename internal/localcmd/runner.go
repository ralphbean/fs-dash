package localcmd

import (
	"context"
	"os/exec"

	"github.com/fullsend-ai/fs-dash/internal/ports"
)

type Runner struct{}

func (Runner) Run(ctx context.Context, command ports.Command) error {
	var cmd *exec.Cmd
	if command.Detach {
		cmd = exec.Command(command.Executable, command.Args...)
	} else {
		cmd = exec.CommandContext(ctx, command.Executable, command.Args...)
	}
	cmd.Dir = command.Dir
	if command.Detach {
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}
	return cmd.Run()
}
