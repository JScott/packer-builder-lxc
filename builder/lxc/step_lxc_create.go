package lxc

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

type stepLxcCreate struct{}

func (s *stepLxcCreate) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	name := config.ContainerName

	// TODO: read from env
	lxc_dir := "/var/lib/lxc"
	rootfs := filepath.Join(lxc_dir, name, "rootfs")

	if config.PackerForce {
		s.Cleanup(state)
	}

	var commands [][]string
	createCommand := append(config.EnvVars, []string{"lxc-create", "-n", name, "-t", config.Name, "--"}...)
	createCommand = append(createCommand, config.Parameters...)
	commands = append(commands, createCommand)
	if len(config.Preload) != 0 {
		tmpPath := "/tmp/lxc-preload/" + name
		commands = append(commands, []string{"rm", "-rf", tmpPath})
		commands = append(commands, []string{"mkdir", "-p", tmpPath})
		// e.g.
		//   [{ "source": "/path/to/some/rootfs.tar.gz", "path": "/" }]
		//   [{ "source": "/path/to/some/rootfs.tar.gz", "extract": "rootfs/tmp", "path": "/tmp" }]
		for _, preload := range config.Preload {
			lxcPath := filepath.Join(rootfs, preload["path"])
			commands = append(commands, []string{"mkdir", "-p", lxcPath})
			tarPath := tmpPath + "/*"
			if val, ok := preload["extract"]; ok {
				tarPath = tmpPath + "/" + val
			}
			commands = append(commands, []string{"tar", "-C", tmpPath, "-xzf", preload["source"]})
			rsyncCommand := "rsync -a " + tarPath + " " + lxcPath
			commands = append(commands, []string{"/bin/sh", "-c", rsyncCommand})
			// find /tmp/jscott/rootfs/tmp/* -maxdepth 1 | xargs -I {} mv {} /tmp/jscott/tmp
			// commands = append(commands, []string{"find", tarPath, "-maxdepth", "1", "|", "xargs", "-I", "{}", "mv", "{}", lxcPath})
		}
		commands = append(commands, []string{"rm", "-rf", tmpPath})
	}
	// prevent tmp from being cleaned on boot, we put provisioning scripts there
	// todo: wait for init to finish before moving on to provisioning instead of this
	commands = append(commands, []string{"touch", filepath.Join(rootfs, "tmp", ".tmpfs")})
	commands = append(commands, []string{"lxc-start", "-d", "--name", name})

	ui.Say("Creating container...")
	for _, command := range commands {
		log.Printf("Executing sudo command: %#v", command)
		err := s.SudoCommand(command...)
		if err != nil {
			err := fmt.Errorf("Error creating container: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	state.Put("mount_path", rootfs)

	return multistep.ActionContinue
}

func (s *stepLxcCreate) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	command := []string{
		"lxc-destroy", "-f", "-n", config.ContainerName,
	}

	ui.Say("Unregistering and deleting virtual machine...")
	if err := s.SudoCommand(command...); err != nil {
		ui.Error(fmt.Sprintf("Error deleting virtual machine: %s", err))
	}
}

func (s *stepLxcCreate) SudoCommand(args ...string) error {
	var stdout, stderr bytes.Buffer

	log.Printf("Executing sudo command: %#v", args)
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stdoutString := strings.TrimSpace(stdout.String())
	stderrString := strings.TrimSpace(stderr.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("Sudo command error: %s", stderrString)
	}

	log.Printf("stdout: %s", stdoutString)
	log.Printf("stderr: %s", stderrString)

	return err
}
