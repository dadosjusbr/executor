package executor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dadosjusbr/executor/status"
)

const (
	noExitError   = -2
	output        = "output"
	dirPermission = 0666
)

// Stage is a phase of data release process.
type Stage struct {
	Name     string            // Stage's name.
	Dir      string            // Directory to be concatenated with default base directory or with the base directory specified here in 'BaseDir'. This field is used to name the image built.
	BaseDir  string            // Base directory for the stage. This field overwrites the DefaultBaseDir in pipeline's definition.
	BuildEnv map[string]string // Variables to be used in the stage build. They will be concatenated with the default variables defined in the pipeline, overwriting them if repeated.
	RunEnv   map[string]string // Variables to be used in the stage run. They will be concatenated with the default variables defined in the pipeline, overwriting them if repeated.
}

// Pipeline represents the sequence of stages for data release.
type Pipeline struct {
	Name            string            // Pipeline's name.
	DefaultBaseDir  string            // Default base directory to be used in all stages.
	DefaultBuildEnv map[string]string // Default variables to be used in the build of all stages.
	DefaultRunEnv   map[string]string // Default variables to be used in the run of all stages.
	Stages          []Stage           // Confguration for the pipeline's stages.
	ErrorHandler    Stage             // Default stage to deal with any errors that occur in the execution of the pipeline.
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
	StartTime   time.Time `json:"start" bson:"start,omitempty"`             // Time at start of stage.
	FinalTime   time.Time `json:"end" bson:"end,omitempty"`                 // Time at the end of stage.
	BuildResult CmdResult `json:"buildResult" bson:"buildResult,omitempty"` // Build result.
	RunResult   CmdResult `json:"runResult" bson:"runResult,omitempty"`     // Run result.
}

// PipelineResult represents the pipeline information and their results.
type PipelineResult struct {
	Name          string                 `json:"name" bson:"name,omitempty"`               // Name of pipeline.
	StagesResults []StageExecutionResult `json:"stageResult" bson:"stageResult,omitempty"` // Results of stage execution.
	StartTime     time.Time              `json:"start" bson:"start,omitempty"`             // Time at start of pipeline.
	FinalTime     time.Time              `json:"final" bson:"final,omitempty"`             // Time at end of pipeline.
	Status        status.Code            `json:"status" bson:"status,omitempty"`           // Pipeline execution status(OK, RunError, BuildError, SetupError).
}

func setup(baseDir string) error {
	finalPath := fmt.Sprintf("%s/%s", baseDir, output)
	if err := os.RemoveAll(finalPath); err != nil {
		return fmt.Errorf("error removing existing output folder: %q", err)
	}

	if err := os.Mkdir(finalPath, dirPermission); err != nil {
		return fmt.Errorf("error creating output folder: %q", err)
	}

	cmdList := strings.Split(fmt.Sprintf("docker volume create --driver local --opt type=none --opt device=%s --opt o=bind --name=dadosjusbr", finalPath), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating volume dadosjusbr: %q", err)
	}

	return nil
}

func tearDown() error {
	cmdList := strings.Split("docker volume rm -f dadosjusbr", " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error removing existing volume dadosjusbr: %q", err)
	}

	return nil
}

// Run executes the pipeline. For each stage defined in the pipeline we execute
// the `docker build` and `docker run`. If any of these two processes fail,
// we interrupt the flow and the error handler is called. Here, we consider a
// failure when the building or execution of the image returns a status other
// than 0 or when an error is raised within the buildImage or runImage functions.
//
// The error handler can be defined as a stage, but it will only be executed
// in case of an error in the standard pipeline flow. If there are any errors
// in the execution of the error handler, the process is stopped and the error is returned.
//
// If a specific error handler has not been defined, the default behavior is to
// return the error message that occurred in the standard flow along with the
// structure that describes all the pipeline execution information.
func (p *Pipeline) Run() (PipelineResult, error) {
	result := PipelineResult{Name: p.Name, StartTime: time.Now()}

	if err := setup(p.DefaultBaseDir); err != nil {
		result.Status = status.SetupError
		return result, fmt.Errorf("error in inicial setup. %q", err)
	}

	for index, stage := range p.Stages {
		var ser StageExecutionResult
		var err error
		ser.Stage = stage.Name
		ser.StartTime = time.Now()

		if len(stage.BaseDir) == 0 {
			stage.BaseDir = p.DefaultBaseDir
		}
		dir := fmt.Sprintf("%s/%s", stage.BaseDir, stage.Dir)

		id := fmt.Sprintf("%s/%s", p.Name, stage.Name)
		// 'index+1' because the index starts from 0.
		log.Printf("Executing Pipeline %s [%d/%d]\n", id, index+1, len(p.Stages))

		stage.BuildEnv = mergeEnv(p.DefaultBuildEnv, stage.BuildEnv)
		ser.BuildResult, err = buildImage(id, dir, stage.BuildEnv)
		if err != nil {
			m := fmt.Sprintf("error when building image: %s", err)
			return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)
		}
		if status.Code(ser.BuildResult.ExitStatus) != status.OK {
			m := fmt.Sprintf("error when building image: status code %d(%s) when building image for %s", ser.BuildResult.ExitStatus, status.Text(status.Code(ser.BuildResult.ExitStatus)), id)
			return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)

		}
		log.Println("Image built sucessfully!")

		stdout := ""
		if index != 0 {
			// 'index-1' is accessing the output from previous stage.
			stdout = result.StagesResults[index-1].RunResult.Stdout
		}

		stage.RunEnv = mergeEnv(p.DefaultRunEnv, stage.RunEnv)
		ser.RunResult, err = runImage(id, dir, stdout, stage.RunEnv)
		if err != nil {
			m := fmt.Sprintf("error when running image: %s", err)
			return handleError(&result, ser, status.RunError, m, p.ErrorHandler)
		}
		if status.Code(ser.RunResult.ExitStatus) != status.OK {
			m := fmt.Sprintf("error when running image: Status code %d(%s) when running image for %s", ser.RunResult.ExitStatus, status.Text(status.Code(ser.RunResult.ExitStatus)), id)
			return handleError(&result, ser, status.RunError, m, p.ErrorHandler)
		}
		log.Printf("Image executed successfully!\n\n")

		ser.FinalTime = time.Now()
		result.StagesResults = append(result.StagesResults, ser)
	}

	if err := tearDown(); err != nil {
		result.Status = status.SetupError
    return result, fmt.Errorf("error in tearDown: %q", err)
	}

	result.Status = status.OK
	result.FinalTime = time.Now()

	return result, nil
}

func handleError(result *PipelineResult, ser StageExecutionResult, previousStatus status.Code, msg string, handler Stage) (PipelineResult, error) {
	ser.FinalTime = time.Now()
	result.StagesResults = append(result.StagesResults, ser)

	if handler.Dir != "" {
		var serError StageExecutionResult
		var err error
		serError.Stage = handler.Name
		serError.StartTime = time.Now()

		id := fmt.Sprintf("%s/%s calls Error Handler", result.Name, ser.Stage)
		serError.BuildResult, err = buildImage(id, handler.Dir, handler.BuildEnv)
		if err != nil {
			result.StagesResults = append(result.StagesResults, serError)
			result.Status = status.ErrorHandlerError
			result.FinalTime = time.Now()

			return *result, fmt.Errorf("error when building image for error handler: %s", err)
		}
		if status.Code(serError.BuildResult.ExitStatus) != status.OK {
			result.StagesResults = append(result.StagesResults, serError)
			result.Status = status.ErrorHandlerError
			result.FinalTime = time.Now()

			return *result, fmt.Errorf("error when building image for error handler: Status code %d(%s) when running image for %s", serError.BuildResult.ExitStatus, status.Text(status.Code(serError.BuildResult.ExitStatus)), id)
		}

		serError.RunResult, err = runImage(id, handler.Dir, string(previousStatus), handler.RunEnv)
		if err != nil {
			result.StagesResults = append(result.StagesResults, serError)
			result.Status = status.ErrorHandlerError
			result.FinalTime = time.Now()

			return *result, fmt.Errorf("error when running image for error handler: %s", err)
		}
		if status.Code(serError.RunResult.ExitStatus) != status.OK {
			result.StagesResults = append(result.StagesResults, serError)
			result.Status = status.ErrorHandlerError
			result.FinalTime = time.Now()

			return *result, fmt.Errorf("error when running image for error handler: Status code %d(%s) when running image for %s", serError.RunResult.ExitStatus, status.Text(status.Code(serError.RunResult.ExitStatus)), id)
		}
	}

	result.Status = previousStatus
	result.FinalTime = time.Now()
	return *result, fmt.Errorf(msg)
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
	switch err.(type) {
	case *exec.Error:
		cmdResultError := CmdResult{
			ExitStatus: statusCode(err),
			Cmd:        strings.Join(cmdList, " "),
		}
		return cmdResultError, fmt.Errorf("command was not executed correctly: %s", err)
	}

	cmdResult := CmdResult{
		Stdout:     string(outb.Bytes()),
		Stderr:     string(errb.Bytes()),
		Cmd:        strings.Join(cmdList, " "),
		CmdDir:     dir,
		ExitStatus: statusCode(err),
		Env:        os.Environ(),
	}

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
func runImage(id, dir, previousStdout string, runEnv map[string]string) (CmdResult, error) {
	log.Printf("Running image for %s", id)

	var builder strings.Builder
	for key, value := range runEnv {
		fmt.Fprintf(&builder, "--env %s=%s ", key, value)
	}
	env := strings.TrimRight(builder.String(), " ")

	cmdList := strings.Split(fmt.Sprintf("docker run -i -v dadosjusbr:/output --rm %s %s", env, filepath.Base(dir)), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(previousStdout)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	log.Printf("$ %s", strings.Join(cmdList, " "))
	err := cmd.Run()
	switch err.(type) {
	case *exec.Error:
		cmdResultError := CmdResult{
			ExitStatus: statusCode(err),
			Cmd:        strings.Join(cmdList, " "),
		}
		return cmdResultError, fmt.Errorf("command was not executed correctly: %s", err)
	}

	cmdResult := CmdResult{
		Stdin:      previousStdout,
		Stdout:     string(outb.Bytes()),
		Stderr:     string(errb.Bytes()),
		Cmd:        strings.Join(cmdList, " "),
		CmdDir:     cmd.Dir,
		ExitStatus: statusCode(err),
		Env:        os.Environ(),
	}

	return cmdResult, nil
}
