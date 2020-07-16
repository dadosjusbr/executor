package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dadosjusbr/executor/status"
)

const noExitError = -2
const output = "output"
const dirPermission = 0644

// Stage is a phase of data release process.
type Stage struct {
	Name     string
	Dir      string
	Repo     string
	BuildEnv map[string]string
	RunEnv   map[string]string
}

// Pipeline represents the sequence of stages for data release.
type Pipeline struct {
	Name            string
	DefaultRepo     string
	DefaultBuildEnv map[string]string
	DefaultRunEnv   map[string]string
	Stages          []Stage
}

// CmdResult represents information about a execution of a command.
type CmdResult struct {
	Stdin      string   `json:"stdin" bson:"stdin,omitempt"`    // String containing the standard input of the process.
	Stdout     string   `json:"stdout" bson:"stdout,omitempty"` // String containing the standard output of the process.
	Stderr     string   `json:"stderr" bson:"stderr,omitempty"` // String containing the standard error of the process.
	Cmd        string   `json:"cmd" bson:"cmd,omitempty"`       // Command that has been executed.
	CmdDir     string   `json:"cmdDir" bson:"cmdir,omitempty"`  // Local directory, in which the command has been executed.
	ExitStatus int      `json:"status" bson:"status,omitempty"` // Exit code of the process executed.
	Env        []string `json:"env" bson:"env,omitempty"`       // Copy of strings representing the environment variables in the form ke=value.
}

// StageExecutionResult represents information about the execution of a stage.
type StageExecutionResult struct {
	Stage       string    `json:"stage" bson:"stage,omitempty"`             // Name of stage.
	StartTime   int64     `json:"start" bson:"start,omitempty"`             // Timestamp at start of stage.
	FinalTime   int64     `json:"end" bson:"end,omitempty"`                 // Timestamp at the end of stage.
	BuildResult CmdResult `json:"buildResult" bson:"buildResult,omitempty"` // Build result.
	RunResult   CmdResult `json:"runResult" bson:"runResult,omitempty"`     // Run result.
}

// PipelineResult represents the pipeline information and their results.
type PipelineResult struct {
	Name          string                 `json:"name" bson:"name,omitempty"`               // Name of pipeline.
	StagesResults []StageExecutionResult `json:"stageResult" bson:"stageResult,omitempty"` // Results of stage execution.
	StartTime     int64                  `json:"start" bson:"start,omitempty"`             // Timestamp at start of pipeline.
	FinalTime     int64                  `json:"final" bson:"final,omitempty"`             // Timestamp at end of pipeline.
	// Todo: checagem e atribuição de status
	Status string `json:"status" bson:"status,omitempty"` // String to inform if the pipepine has finished with sucess or not.
}

func setup(repo, dir string) error {
	finalPath := fmt.Sprintf("%s/%s/%s", repo, dir, output)
	if os.IsNotExist(os.Mkdir(finalPath, dirPermission)) {
		if err := os.Mkdir(finalPath, os.ModeDir); err != nil {
			return fmt.Errorf("error creating output folder: %q", err)
		}
	}

	cmdList := strings.Split(fmt.Sprintf("docker volume create --driver local --opt type=none --opt device=%s --opt o=bind --name=dadosjusbr", finalPath), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating volume dadosjusbr: %q", err)
	}

	return nil
}

// Run executes the pipeline
func (p *Pipeline) Run() (PipelineResult, error) {
	result := PipelineResult{Name: p.Name, StartTime: time.Now().Unix()}

	for index, stage := range p.Stages {
		var ser StageExecutionResult
		var err error

		if len(stage.Repo) == 0 {
			stage.Repo = p.DefaultRepo
		}
		if err := setup(stage.Repo, stage.Dir); err != nil {
			return PipelineResult{}, fmt.Errorf("error in inicial setup. %q", err)
		}

		id := fmt.Sprintf("%s/%s", p.Name, stage.Name)
		log.Printf("Executing Pipeline %s [%d/%d]\n", id, index+1, len(p.Stages))

		ser.Stage = stage.Name
		ser.StartTime = time.Now().Unix()
		dir := fmt.Sprintf("%s/%s", stage.Repo, stage.Dir)

		stage.BuildEnv = mergeEnv(p.DefaultBuildEnv, stage.BuildEnv)
		ser.BuildResult, err = buildImage(id, dir, stage.BuildEnv)
		if err != nil {
			storeError("error when building image", err)
		}

		stdout := ""
		if index != 0 {
			stdout = result.StagesResults[index-1].RunResult.Stdout
		}

		stage.RunEnv = mergeEnv(p.DefaultRunEnv, stage.RunEnv)
		ser.RunResult, err = runImage(id, dir, stdout, stage.RunEnv)
		if err != nil {
			storeError("error when running image", err)
		}

		ser.FinalTime = time.Now().Unix()
		result.StagesResults = append(result.StagesResults, ser)
	}

	return result, nil
}

func storeError(msg string, err error) error {
	return fmt.Errorf("%s - %s", msg, err)
	// TODO: Store error
	//er.Cr.AgencyID = filepath.Base(job)
	//Store Error
	//Build(storeErrDir, commit, conf)
	//execStoreErr(er, conf)
	//continue
}

func mergeEnv(defaultEnv, stageEnv map[string]string) map[string]string {
	env := make(map[string]string)

	for k, v := range defaultEnv {
		env[k] = v
	}
	for k, v := range stageEnv {
		env[k] = v
	}
	return env
}

func buildImage(id, dir string, buildEnv map[string]string) (CmdResult, error) {
	log.Printf("Building image for %s", id)

	var b strings.Builder
	for k, v := range buildEnv {
		fmt.Fprintf(&b, "--build-arg %s=%s ", k, v)
	}
	env := b.String()

	cmdList := strings.Split(fmt.Sprintf("docker build %s-t %s .", env, filepath.Base(dir)), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	cmd.Dir = dir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	log.Printf("$ %s", strings.Join(cmdList, " "))
	err := cmd.Run()
	exitStatus := statusCode(err)

	if status.Code(exitStatus) != status.OK {
		cmdResultError := CmdResult{
			Stderr:     string(errb.Bytes()),
			Cmd:        strings.Join(cmdList, " "),
			CmdDir:     dir,
			ExitStatus: exitStatus,
		}
		return cmdResultError, fmt.Errorf("Status code %d(%s) when building image for %s", exitStatus, status.Text(status.Code(exitStatus)), id)
	}

	cmdResult := CmdResult{
		Stdout:     string(outb.Bytes()),
		Stderr:     string(errb.Bytes()),
		Cmd:        strings.Join(cmdList, " "),
		CmdDir:     dir,
		ExitStatus: exitStatus,
		Env:        os.Environ(),
	}
	log.Println("Image build sucessfully!")

	return cmdResult, nil
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
	return noExitError
}

// runImage executes the image designed and returns it's stdin, stdout and exit error if any.
func runImage(id, dir, stdout string, runEnv map[string]string) (CmdResult, error) {
	log.Printf("Running image for %s", id)

	stdoutJSON, err := json.Marshal(stdout)
	if err != nil {
		return CmdResult{}, fmt.Errorf("Error trying to marshal stage execution result %s", stdoutJSON)
	}

	var builder strings.Builder
	for key, value := range runEnv {
		fmt.Fprintf(&builder, "%s=%s ", key, value)
	}
	env := strings.TrimRight(builder.String(), " ")

	cmdList := strings.Split(fmt.Sprintf("docker run -i -v dadosjusbr:/output --rm %s %s", filepath.Base(dir), env), " ")
	cmd := exec.Command("docker", cmdList...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(string(stdoutJSON))
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	log.Printf("$ %s", strings.Join(cmdList, " "))
	err = cmd.Run()
	exitStatus := statusCode(err)

	if status.Code(exitStatus) != status.OK {
		cmdResultError := CmdResult{
			Stderr:     string(errb.Bytes()),
			Cmd:        strings.Join(cmdList, " "),
			CmdDir:     cmd.Dir,
			ExitStatus: exitStatus,
		}
		return cmdResultError, fmt.Errorf("Status code %d(%s) when running image for %s", exitStatus, status.Text(status.Code(exitStatus)), id)
	}
	cmdResult := CmdResult{
		Stdin:      string(stdoutJSON),
		Stdout:     string(outb.Bytes()),
		Stderr:     string(errb.Bytes()),
		Cmd:        strings.Join(cmdList, " "),
		CmdDir:     cmd.Dir,
		ExitStatus: exitStatus,
		Env:        os.Environ(),
	}
	log.Printf("%s executed successfully\n\n", id)

	return cmdResult, nil
}
