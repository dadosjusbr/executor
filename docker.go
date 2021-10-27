package executor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// buildImage executes the 'docker build' for a image, considering the
// parameters defined for it and returns a CmdResult and an error, if any.
func buildImage(id, baseDir, stageDir string, env map[string]string) (CmdResult, error) {
	dir := filepath.Join(baseDir, stageDir)
	var b strings.Builder
	for k, v := range env {
		fmt.Fprintf(&b, "--build-arg %s=%s ", k, fmt.Sprintf(`"%s"`, v))
	}
	envStr := b.String()

	cmdStr := fmt.Sprintf("docker build %s-t %s .", envStr, id)
	// sh -c is a workaround that allow us to have double quotes around environment variable values.
	// Those are needed when the environment variables have whitespaces, for instance a NAME, like in
	// TREPB.
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = dir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	log.Printf("$ %s", cmdStr)
	err := cmd.Run()
	switch err.(type) {
	case *exec.Error:
		cmdResultError := CmdResult{
			ExitStatus: statusCode(err),
			Cmd:        cmdStr,
		}
		return cmdResultError, fmt.Errorf("command was not executed correctly: %s", err)
	}

	cmdResult := CmdResult{
		Stdout:     outb.String(),
		Stderr:     errb.String(),
		Cmd:        cmdStr,
		CmdDir:     dir,
		ExitStatus: statusCode(err),
		Env:        os.Environ(),
	}

	return cmdResult, err
}

// runImage executes the 'docker run' for a image, considering the
// parameters defined for it and returns a CmdResult and an error, if any.
// It uses the stdout from the previous stage as the stdin for this new command.
func runImage(id, baseDir, stageDir, stdin string, env map[string]string) (CmdResult, error) {
	dir := filepath.Join(baseDir, stageDir)
	var builder strings.Builder
	for key, value := range env {
		fmt.Fprintf(&builder, "--env %s=%s ", key, fmt.Sprintf(`"%s"`, value))
	}
	envStr := strings.TrimRight(builder.String(), " ")

	cmdStr := fmt.Sprintf("docker run -i -v dadosjusbr:/output --rm %s %s", envStr, id)
	// sh -c is a workaround that allow us to have double quotes around environment variable values.
	// Those are needed when the environment variables have whitespaces, for instance a NAME, like in
	// TREPB.
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	log.Printf("$ %s", cmdStr)
	err := cmd.Run()
	switch err.(type) {
	case *exec.Error:
		cmdResultError := CmdResult{
			ExitStatus: statusCode(err),
			Cmd:        cmdStr,
		}
		return cmdResultError, fmt.Errorf("command was not executed correctly: %s", err)
	}

	cmdResult := CmdResult{
		Stdin:      stdin,
		Stdout:     outb.String(),
		Stderr:     errb.String(),
		Cmd:        cmdStr,
		CmdDir:     cmd.Dir,
		ExitStatus: statusCode(err),
		Env:        os.Environ(),
	}

	return cmdResult, err
}

func createVolume(dir string) error {
	baseDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory:%v", err)
	}
	cmdList := strings.Split(fmt.Sprintf("docker volume create --driver local --opt type=none --opt device=%s --opt o=bind --name=dadosjusbr", dir), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	cmd.Dir = baseDir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	log.Printf("$ %s", strings.Join(cmdList, " "))
	switch cmd.Run().(type) {
	case *exec.Error:
		r := CmdResult{
			ExitStatus: statusCode(err),
			Cmd:        strings.Join(cmdList, " "),
		}
		return fmt.Errorf("command was not executed correctly: %+v", r)
	}
	return nil
}

func removeVolume(dir string) error {
	cmdList := strings.Split("docker volume rm -f dadosjusbr", " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	log.Printf("$ %s", strings.Join(cmdList, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error removing existing volume dadosjusbr: %q", err)
	}
	return nil
}
