package lxc

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
	"github.com/mitchellh/multistep"
)

// The unique ID for this builder
const BuilderId = "ustream.lxc"

type wrappedCommandTemplate struct {
	Command string
}

type Builder struct {
	config *Config
	runner multistep.Runner
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	c, errs := NewConfig(raws...)
	if errs != nil {
		return nil, errs
	}
	b.config = c

	return nil, nil
}

func (b *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("The lxc builder only works on linux environments.")
	}

	wrappedCommand := func(command string) (string, error) {
		b.config.ctx.Data = &wrappedCommandTemplate{Command: command}
		return interpolate.Render(b.config.CommandWrapper, &b.config.ctx)
	}

	steps := []multistep.Step{
		new(stepPrepareOutputDir),
		new(stepLxcCreate),
		&StepWaitInit{
			WaitTimeout: b.config.InitTimeout,
		},
		new(StepProvision),
		new(stepExport),
	}

	// Setup the state bag
	state := new(multistep.BasicStateBag)
	state.Put("config", b.config)
	state.Put("cache", cache)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("wrappedCommand", CommandWrapper(wrappedCommand))

	// Run
	if b.config.PackerDebug {
		b.runner = &multistep.DebugRunner{
			Steps:   steps,
			PauseFn: common.MultistepDebugFn(ui),
		}
	} else {
		b.runner = &multistep.BasicRunner{Steps: steps}
	}

	b.runner.Run(state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	// Compile the artifact list
	files := make([]string, 0, 5)
	visit := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}

		return err
	}

	if err := filepath.Walk(b.config.OutputDir, visit); err != nil {
		return nil, err
	}

	artifact := &Artifact{
		dir: b.config.OutputDir,
		f:   files,
	}

	return artifact, nil
}

func (b *Builder) Cancel() {
	if b.runner != nil {
		log.Println("Cancelling the step runner...")
		b.runner.Cancel()
	}
}
