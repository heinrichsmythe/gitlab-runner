package ssh

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/ssh"
)

type executor struct {
	executors.AbstractExecutor
	sshCommand ssh.Client
}

func (s *executor) Prepare(options common.ExecutorPrepareOptions) error {
	err := s.AbstractExecutor.Prepare(options)
	if err != nil {
		return err
	}

	s.Println("Using SSH executor...")
	if s.BuildShell.PassFile {
		return errors.New("SSH doesn't support shells that require script file")
	}

	if s.Config.SSH == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Debugln("Starting SSH command...")

	// Create SSH command
	s.sshCommand = ssh.Client{
		Config: *s.Config.SSH,
		Stdout: s.Trace,
		Stderr: s.Trace,
	}

	s.Debugln("Connecting to SSH server...")
	err = s.sshCommand.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	err := s.sshCommand.Run(cmd.Context, ssh.Command{
		Environment: s.BuildShell.Environment,
		Command:     s.BuildShell.GetCommandWithArguments(),
		Stdin:       cmd.Script,
	})
	if _, ok := err.(*ssh.ExitError); ok {
		err = &common.BuildError{Inner: err}
	}
	return err
}

func (s *executor) Cleanup() {
	s.sshCommand.Cleanup()
	s.AbstractExecutor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultCustomBuildsDirEnabled: false,
		DefaultBuildsDir:              "builds",
		SharedBuildsDir:               true,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.LoginShell,
			RunnerCommand: "gitlab-runner",
		},
		ShowHostname: true,
	}

	creator := func() common.Executor {
		return &executor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
		features.Shared = true
	}

	common.RegisterExecutor("ssh", executors.DefaultExecutorProvider{
		Creator:          creator,
		FeaturesUpdater:  featuresUpdater,
		DefaultShellName: options.Shell.Shell,
	})
}
