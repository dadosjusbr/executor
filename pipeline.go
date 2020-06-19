package executor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dadosjusbr/coletores/status"
)

//Stage is a phase of data release process.
type Stage struct {
	Name string
	Dir  string
	Repo string
	Env  map[string]string
}

//Pipeline represents the sequence of stages for data release.
type Pipeline struct {
	Name        string
	DefaultRepo string
	DefaultEnv  map[string]string
	Stages      []Stage
}

// StageExecutionResult represents information about the execution of a stage.
type StageExecutionResult struct {
	Stdin      string   `json:"stdin" bson:"stdin,omitempty"`             // String containing the standard input of the process.
	Stdout     string   `json:"stdout" bson:"stdout,omitempty"`           // String containing the standard output of the process.
	Stderr     string   `json:"stderr" bson:"stderr,omitempty"`           // String containing the standard error of the process.
	Cmd        string   `json:"cmd" bson:"cmd,omitempty"`                 // Command that has been executed
	CmdDir     string   `json:"cmddir" bson:"cmdir,omitempty"`            // Local directory, in which the command has been executed
	ExitStatus int      `json:"status,omitempty" bson:"status,omitempty"` // Exit code of the process executed
	Env        []string `json:"env,omitempty" bson:"env,omitempty"`       // Copy of strings representing the environment variables in the form ke=value
}

func setup(path string) error {
	if os.IsNotExist(os.Mkdir(path+"/output", os.ModeDir)) {
		if err := os.Mkdir(path+"/output", os.ModeDir); err != nil {
			return fmt.Errorf("error creating output folder: %q", err)
		}
	}

	cmdList := strings.Split(fmt.Sprintf("docker volume create --driver local --opt type=none --opt device=%s/output --opt o=bind --name=dadosjusbr", path), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating volume dadosjusbr: %q", err)
	}

	return nil
}

//Run executes the pipeline
func (p *Pipeline) Run(pipeline Pipeline) ([]StageExecutionResult, error) {
	if err := setup(pipeline.DefaultRepo); err != nil {
		return nil, fmt.Errorf("error in inicial setup. %q", err)
	}

	for _, stage := range pipeline.Stages {
		var er StageExecutionResult
		var err error

		id := fmt.Sprintf("%s-%s-%s", stage.Name, pipeline.DefaultEnv["month"], pipeline.DefaultEnv["year"])
		log.Printf("Executing %s ...\n", id)

		er, err = build(stage)
		if err != nil {
			return nil, fmt.Errorf("error in image build %s", err)
		}

		fmt.Println(er)
	}

	return []StageExecutionResult{}, nil
}

// Build tries to build a docker image for a job and panics if it can not suceed.
func build(s Stage) (StageExecutionResult, error) {
	id := fmt.Sprintf("%s", s.Name)
	log.Printf("Building image %s...\n", id)

	pi, err := buildImage(s.Dir, s.Env["commit"])
	if err != nil {
		return *pi, fmt.Errorf("Error building DataCollector image %s: %q", id, err)
	} else if status.Code(pi.ExitStatus) != status.OK {
		return *pi, fmt.Errorf("Status code %d(%s) building DataCollector image %s", pi.ExitStatus, status.Text(status.Code(pi.ExitStatus)), id)
	}
	log.Printf("Image %s build sucessfully\n", id)
	return *pi, nil
}

// build runs a go build for each path. It will also insert the value of main.gitCommit in the binaries.
func buildImage(dir, commit string) (*StageExecutionResult, error) {
	cmdList := strings.Split(fmt.Sprintf("docker build --build-arg GIT_COMMIT=%s -t %s .", commit, filepath.Base(dir)), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	cmd.Dir = dir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	exitStatus := statusCode(err)
	procInfo := StageExecutionResult{
		Stdout:     string(outb.Bytes()),
		Stderr:     string(errb.Bytes()),
		Cmd:        strings.Join(cmdList, " "),
		CmdDir:     dir,
		ExitStatus: exitStatus,
		Env:        os.Environ(),
	}
	return &procInfo, err
}

// statusCode returns the exit code returned for the cmd execution.
// 0 if no error.
// -1 if process was terminated by a signal or hasn't started.
// -2 if error is not an ExitError.
func statusCode(err error) int {
	if err == nil {
		return 0
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return -2
}
