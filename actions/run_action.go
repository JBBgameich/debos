/*
Run Action

Allows to run any available command or script in the filesystem or
in host environment.

Yaml syntax:
 - action: run
   chroot: bool
   postprocess: bool
   script: script name
   command: command line

Properties 'command' and 'script' are mutually exclusive.

- command -- command with arguments; the command expected to be accessible in
host's or chrooted environment -- depending on 'chroot' property.

- script -- script with arguments; script must be located in recipe directory.

Optional properties:

- chroot -- run script or command in target filesystem if set to true.
In other case the command or script is executed within the build process, with
access to the filesystem and the image. In both cases it is run with root privileges.

- postprocess -- if set script or command is executed after all other commands and
has access to the image file.


Properties 'chroot' and 'postprocess' are mutually exclusive.
*/
package actions

import (
	"errors"
	"fmt"
	"github.com/sjoerdsimons/fakemachine"
	"path"

	"github.com/go-debos/debos"
)

type RunAction struct {
	debos.BaseAction `yaml:",inline"`
	Chroot           bool
	PostProcess      bool
	Script           string
	Command          string
}

func (run *RunAction) Verify(context *debos.DebosContext) error {
	if run.PostProcess && run.Chroot {
		return errors.New("Cannot run postprocessing in the chroot")
	}
	return nil
}

func (run *RunAction) PreMachine(context *debos.DebosContext, m *fakemachine.Machine,
	args *[]string) error {

	if run.Script == "" {
		return nil
	}

	run.Script = debos.CleanPathAt(run.Script, context.RecipeDir)
	if !run.PostProcess {
		m.AddVolume(path.Dir(run.Script))
	}

	return nil
}

func (run *RunAction) doRun(context debos.DebosContext) error {
	run.LogStart()
	var cmdline []string
	var label string
	var cmd debos.Command

	if run.Chroot {
		cmd = debos.NewChrootCommand(context.Rootdir, context.Architecture)
	} else {
		cmd = debos.Command{}
	}

	if run.Script != "" {
		run.Script = debos.CleanPathAt(run.Script, context.RecipeDir)
		if run.Chroot {
			cmd.AddBindMount(path.Dir(run.Script), "/script")
			cmdline = []string{fmt.Sprintf("/script/%s", path.Base(run.Script))}
		} else {
			cmdline = []string{run.Script}
		}
		label = path.Base(run.Script)
	} else {
		cmdline = []string{"sh", "-c", run.Command}
		label = run.Command
	}

	if !run.Chroot && !run.PostProcess {
		cmd.AddEnvKey("ROOTDIR", context.Rootdir)
	}

	return cmd.Run(label, cmdline...)
}

func (run *RunAction) Run(context *debos.DebosContext) error {
	if run.PostProcess {
		/* This runs in postprocessing instead */
		return nil
	}
	return run.doRun(*context)
}

func (run *RunAction) PostMachine(context debos.DebosContext) error {
	if !run.PostProcess {
		return nil
	}
	return run.doRun(context)
}