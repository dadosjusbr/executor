package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/dadosjusbr/executor/status"
	"github.com/go-git/go-git/v5"
)

const (
	noExitError   = -2
	output        = "output"
	dirPermission = 0666
)

// Stage is a phase of data release process.
type Stage struct {
	Name     string            `json:"name" bson:"name,omitempt"`           // Stage's name.
	Dir      string            `json:"dir" bson:"dir,omitempt"`             // Directory to be concatenated with default base directory or with the base directory specified here in 'BaseDir'. This field is used to name the image built.
	Repo     string            `json:"repo" bson:"repo,omitempt"`           // Repository URL from where to clone the pipeline stage.
	BaseDir  string            `json:"base-dir" bson:"base-dir,omitempt"`   // Base directory for the stage. This field overwrites the DefaultBaseDir in pipeline's definition.
	BuildEnv map[string]string `json:"build-env" bson:"build-env,omitempt"` // Variables to be used in the stage build. They will be concatenated with the default variables defined in the pipeline, overwriting them if repeated.
	RunEnv   map[string]string `json:"run-env" bson:"run-env,omitempt"`     // Variables to be used in the stage run. They will be concatenated with the default variables defined in the pipeline, overwriting them if repeated.
}

// Pipeline represents the sequence of stages for data release.
type Pipeline struct {
	Name            string            `json:"name" bson:"name,omitempt"`                           // Pipeline's name.
	DefaultBaseDir  string            `json:"default-base-dir" bson:"default-base-dir,omitempt"`   // Default base directory to be used in all stages.
	DefaultBuildEnv map[string]string `json:"default-build-env" bson:"default-build-env,omitempt"` // Default variables to be used in the build of all stages.
	DefaultRunEnv   map[string]string `json:"default-run-env" bson:"default-run-env,omitempt"`     // Default variables to be used in the run of all stages.
	Stages          []Stage           `json:"stages" bson:"stages,omitempt"`                       // Confguration for the pipeline's stages.
	ErrorHandler    Stage             `json:"error-handler" bson:"error-handler,omitempt"`         // Default stage to deal with any errors that occur in the execution of the pipeline.
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
	Stage       Stage     `json:"stage" bson:"stage,omitempty"`             // Name of stage.
	CommitID    string    `json:"commit" bson:"commit,omitempty"`           // Commit of the stage repo when executing the stage.
	StartTime   time.Time `json:"start" bson:"start,omitempty"`             // Time at start of stage.
	FinalTime   time.Time `json:"end" bson:"end,omitempty"`                 // Time at the end of stage.
	BuildResult CmdResult `json:"buildResult" bson:"buildResult,omitempty"` // Build result.
	RunResult   CmdResult `json:"runResult" bson:"runResult,omitempty"`     // Run result.
}

// PipelineResult represents the pipeline information and their results.
type PipelineResult struct {
	Name         string                 `json:"name" bson:"name,omitempty"`               // Name of pipeline.
	StageResults []StageExecutionResult `json:"stageResult" bson:"stageResult,omitempty"` // Results of stage execution.
	StartTime    time.Time              `json:"start" bson:"start,omitempty"`             // Time at start of pipeline.
	FinalTime    time.Time              `json:"final" bson:"final,omitempty"`             // Time at end of pipeline.
	Status       string                 `json:"status" bson:"status,omitempty"`           // Pipeline execution status(OK, RunError, BuildError, SetupError...).
}

func setup(baseDir string) (CmdResult, error) {
	finalPath := fmt.Sprintf("%s/%s", baseDir, output)
	log.Printf("$ rm -rf %s", finalPath)
	if err := os.RemoveAll(finalPath); err != nil {
		return CmdResult{
			Stdout:     "",
			Stderr:     err.Error(),
			Cmd:        fmt.Sprintf("rm -rf %s", finalPath),
			CmdDir:     baseDir,
			ExitStatus: 1,
			Env:        os.Environ(),
		}, nil
	}
	log.Printf("$ mkdir -m %d %s", dirPermission, finalPath)
	if err := os.Mkdir(finalPath, dirPermission); err != nil {
		return CmdResult{
			Stdout:     "",
			Stderr:     err.Error(),
			Cmd:        fmt.Sprintf("mkdir -m %d %s", dirPermission, finalPath),
			CmdDir:     baseDir,
			ExitStatus: 1,
			Env:        os.Environ(),
		}, nil
	}

	cmdList := strings.Split(fmt.Sprintf("docker volume create --driver local --opt type=none --opt device=%s --opt o=bind --name=dadosjusbr", finalPath), " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	cmd.Dir = baseDir
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
	return CmdResult{
		Stdout:     outb.String(),
		Stderr:     errb.String(),
		Cmd:        strings.Join(cmdList, " "),
		CmdDir:     baseDir,
		ExitStatus: statusCode(err),
		Env:        os.Environ(),
	}, nil
}

func tearDown() error {
	cmdList := strings.Split("docker volume rm -f dadosjusbr", " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error removing existing volume dadosjusbr: %q", err)
	}

	return nil
}

// Run executes the pipeline.
// For each stage defined in the pipeline we execute the `docker build` and
// `docker run`. If any of these two processes fail, we interrupt the flow
// and the error handler is called. Here, we consider a failure when the
// building or execution of the image returns a status other than 0 or
// when an error is raised within the buildImage or runImage functions.
//
// The error handler can be defined as a stage, but it will only be executed in
// case of an error in the pipeline standard flow, which is when we call the
// function handleError.
//
// When handleError is called we pass all informations about the pipeline
// execution until that point, which are:
// - the PipelineResult (until current stage),
// - the StageResult (from current stage),
// - error status and error message
// - the stage ErrorHandler (defined in the Pipeline).
// Thereby the error handler will able to process or store the problem that
// occurred in the current stage. The function runImage for stage ErrorHandler
// receives the StageResult as STDIN.
// If there are any errors in the execution of the error handler,
// the processing is completely stopped and the error is returned.
//
// If a specific error handler has not been defined, the default behavior is to
// return the error message that occurred in the standard flow along with the
// structure that describes all the pipeline execution information until that point.
func (p *Pipeline) Run() (PipelineResult, error) {
	result := PipelineResult{Name: p.Name, StartTime: time.Now()}

	log.Printf("Setting up Pipeline %s\n", p.Name)
	setupRes, err := setup(p.DefaultBaseDir)
	if err != nil {
		result.Status = status.Text(status.SetupError)
		return result, status.NewError(status.SetupError, fmt.Errorf("error in inicial setup: %q", err))
	}
	if status.Code(setupRes.ExitStatus) != status.OK {
		// Treating setup as special stage result to enjoy the error treatment.
		m := fmt.Sprintf("error setting up pipeline: status code %d(%s) when setting up pipeline", setupRes.ExitStatus, status.Text(status.Code(setupRes.ExitStatus)))
		return handleError(&result, StageExecutionResult{
			Stage:     Stage{Name: "Pipeline_Setup"},
			RunResult: setupRes,
		}, status.SetupError, m, p.ErrorHandler)
	}

	for index, stage := range p.Stages {
		var ser StageExecutionResult
		var err error
		ser.Stage = stage
		ser.StartTime = time.Now()
		if stage.BaseDir == "" {
			stage.BaseDir = p.DefaultBaseDir
		}
		id := fmt.Sprintf("%s/%s", p.Name, stage.Name)
		idContainer := strings.ReplaceAll(strings.ToLower(stage.Name), " ", "-")
		// 'index+1' because the index starts from 0.
		log.Printf("\nExecuting Pipeline Stage %s [%d/%d]\n", id, index+1, len(p.Stages))

		// if there the field "repo" is set for the stage, clone it and update
		// its baseDir and commit id.
		if stage.Repo != "" {
			u, err := url.Parse(stage.Repo)
			if err != nil {
				m := fmt.Sprintf("error parsing repository URL: %s", err)
				return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)
			}
			if u.Scheme == "" {
				u.Scheme = "https"
			}
			log.Printf("Cloning repo %s\n", u.String())
			// spaces are super bad for paths in command-line
			tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("dadosjusbr-executor-%s", path.Base(u.Path)))
			if err := os.MkdirAll(tmpDir, 0775); err != nil {
				m := fmt.Sprintf("error when creating temporary dir: %s", err)
				return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)
			}
			cid, err := cloneRepository(tmpDir, u.String())
			if err != nil {
				m := fmt.Sprintf("error when cloning repo: %s", err)
				return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)
			}
			ser.CommitID = cid
			stage.BaseDir = path.Join(tmpDir)
			log.Printf("Repo cloned successfully! Commit:%s New dir:%s\n", ser.CommitID, stage.BaseDir)
		}

		stage.BuildEnv = mergeEnv(p.DefaultBuildEnv, stage.BuildEnv)
		ser.BuildResult, err = buildImage(idContainer, stage.BaseDir, stage.Dir, stage.BuildEnv)
		if err != nil {
			m := fmt.Sprintf("error when building image: %s", err)
			return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)
		}
		if status.Code(ser.BuildResult.ExitStatus) != status.OK {
			m := fmt.Sprintf("error when building image: status code %d(%s) when building image for %s", ser.BuildResult.ExitStatus, status.Text(status.Code(ser.BuildResult.ExitStatus)), id)
			return handleError(&result, ser, status.BuildError, m, p.ErrorHandler)

		}
		log.Printf("Image built sucessfully!\n\n")

		stdout := ""
		if index == 0 {
			in, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Printf("Erro lendo dados da entrada padrão: %q. Continuando...", err)
			} else {
				stdout = string(in)
			}
		} else {
			// 'index-1' is accessing the output from previous stage.
			stdout = result.StageResults[index-1].RunResult.Stdout
		}

		stage.RunEnv = mergeEnv(p.DefaultRunEnv, stage.RunEnv)
		ser.RunResult, err = runImage(idContainer, stage.BaseDir, stage.Dir, stdout, stage.RunEnv)
		if err != nil {
			m := fmt.Sprintf("error when running image: %s", err)
			return handleError(&result, ser, status.RunError, m, p.ErrorHandler)
		}
		if status.Code(ser.RunResult.ExitStatus) != status.OK {
			m := fmt.Sprintf("error when running image: Status code %d(%s) when running image for %s", ser.RunResult.ExitStatus, status.Text(status.Code(ser.RunResult.ExitStatus)), id)
			return handleError(&result, ser, status.RunError, m, p.ErrorHandler)
		}
		log.Printf("Image executed successfully!\n")

		ser.FinalTime = time.Now()

		// Removing temporary directories created from cloned repositories.
		if stage.Repo != "" {
			log.Printf("Removing cloned repo %s\n", stage.BaseDir)
			if err := os.RemoveAll(stage.BaseDir); err != nil {
				result.Status = status.Text(status.SystemError)
				return result, status.NewError(status.SystemError, fmt.Errorf("error removing temp dir(%s): %q", stage.BaseDir, err))
			}
			log.Printf("Cloned repo temp dir removed successfully!\n\n")
		}
		fmt.Printf("\n")
		result.StageResults = append(result.StageResults, ser)
	}

	if err := tearDown(); err != nil {
		result.Status = status.Text(status.SetupError)
		return result, status.NewError(status.SetupError, fmt.Errorf("error in tear down: %q", err))
	}

	result.Status = status.Text(status.OK)
	result.FinalTime = time.Now()

	return result, nil
}

// handleError is responsible for build and run the stage ErrorHandler
// defined in the Pipeline. It is called when occurs any error in
// pipeline standard flow. If a specific error handler has not been
// defined, the default behavior is to return the PipelineResult until
// the last stage executed and the error occurred.
//
// When handleError is called it receives all informations about the pipeline
// execution until that point, which are:
// - the PipelineResult (until current stage),
// - the StageResult (from current stage),
// - error status and error message
// - the stage ErrorHandler (defined in the Pipeline).
// Thereby the error handler will able to process or store the problem that
// occurred in the current stage. The function runImage for stage ErrorHandler
// receives the StageResult as STDIN.
//
// If there are any errors in the execution of the error handler,
// the processing is completely stopped and the error is returned.
func handleError(result *PipelineResult, previousSer StageExecutionResult, previousStatus status.Code, msg string, handler Stage) (PipelineResult, error) {
	if reflect.ValueOf(handler).IsZero() {
		erStdin, err := json.MarshalIndent(previousSer, "", "\t")
		if err != nil {
			return *result, status.NewError(status.ErrorHandlerError, fmt.Errorf("error marshaling input of error handler: %s", err))
		}
		fmt.Println(string(erStdin))
		return *result, fmt.Errorf(msg)
	}
	previousSer.FinalTime = time.Now()
	result.StageResults = append(result.StageResults, previousSer)

	if handler.Dir != "" {
		var serError StageExecutionResult
		var err error
		serError.Stage = handler
		serError.StartTime = time.Now()

		id := fmt.Sprintf("%s/%s calls Error Handler", result.Name, previousSer.Stage)
		serError.BuildResult, err = buildImage(id, handler.BaseDir, handler.Dir, handler.BuildEnv)
		if err != nil {
			result.StageResults = append(result.StageResults, serError)
			result.Status = status.Text(status.ErrorHandlerError)
			result.FinalTime = time.Now()

			return *result, status.NewError(status.BuildError, fmt.Errorf("error when building image for error handler: %s", err))
		}
		if status.Code(serError.BuildResult.ExitStatus) != status.OK {
			result.StageResults = append(result.StageResults, serError)
			result.Status = status.Text(status.ErrorHandlerError)
			result.FinalTime = time.Now()

			return *result, status.NewError(status.BuildError, fmt.Errorf("error when building image for error handler: Status code %d(%s) when running image for %s", serError.BuildResult.ExitStatus, status.Text(status.Code(serError.BuildResult.ExitStatus)), id))
		}

		erStdin, err := json.Marshal(previousSer)
		if err != nil {
			return *result, status.NewError(status.ErrorHandlerError, fmt.Errorf("error in parser StageExecutionResult for error handler: %s", err))
		}
		serError.RunResult, err = runImage(id, handler.BaseDir, handler.Dir, string(erStdin), handler.RunEnv)
		if err != nil {
			result.StageResults = append(result.StageResults, serError)
			result.Status = status.Text(status.ErrorHandlerError)
			result.FinalTime = time.Now()

			return *result, status.NewError(status.RunError, fmt.Errorf("error when running image for error handler: %s", err))
		}
		if status.Code(serError.RunResult.ExitStatus) != status.OK {
			result.StageResults = append(result.StageResults, serError)
			result.Status = status.Text(status.ErrorHandlerError)
			result.FinalTime = time.Now()

			return *result, status.NewError(status.RunError, fmt.Errorf("error when running image for error handler: Status code %d(%s) when running image for %s", serError.RunResult.ExitStatus, status.Text(status.Code(serError.RunResult.ExitStatus)), id))
		}
	}

	result.Status = status.Text(previousStatus)
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

// buildImage executes the 'docker build' for a image, considering the
// parameters defined for it and returns a CmdResult and an error, if any.
func buildImage(id, baseDir, stageDir string, buildEnv map[string]string) (CmdResult, error) {
	log.Printf("Building image for %s", id)

	dir := filepath.Join(baseDir, stageDir)
	var b strings.Builder
	for k, v := range buildEnv {
		fmt.Fprintf(&b, "--build-arg %s=%s ", k, fmt.Sprintf(`"%s"`, v))
	}
	env := b.String()

	cmdStr := fmt.Sprintf("docker build %s-t %s .", env, id)
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

// runImage executes the 'docker run' for a image, considering the
// parameters defined for it and returns a CmdResult and an error, if any.
// It uses the stdout from the previous stage as the stdin for this new command.
func runImage(id, baseDir, stageDir, previousStdout string, runEnv map[string]string) (CmdResult, error) {
	log.Printf("Running image for %s", id)
	dir := filepath.Join(baseDir, stageDir)
	var builder strings.Builder
	for key, value := range runEnv {
		fmt.Fprintf(&builder, "--env %s=%s ", key, fmt.Sprintf(`"%s"`, value))
	}
	env := strings.TrimRight(builder.String(), " ")

	cmdStr := fmt.Sprintf("docker run -i -v dadosjusbr:/output --rm %s %s", env, id)
	// sh -c is a workaround that allow us to have double quotes around environment variable values.
	// Those are needed when the environment variables have whitespaces, for instance a NAME, like in
	// TREPB.
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(previousStdout)
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
		Stdin:      previousStdout,
		Stdout:     outb.String(),
		Stderr:     errb.String(),
		Cmd:        cmdStr,
		CmdDir:     cmd.Dir,
		ExitStatus: statusCode(err),
		Env:        os.Environ(),
	}

	return cmdResult, err
}

// cloneRepository is responsible for get the latest code version of pipeline repository.
// Creates and returns the DefaultBaseDir for the pipeline and the latest commit in the repository.
func cloneRepository(defaultBaseDir, url string) (string, error) {
	if err := os.RemoveAll(defaultBaseDir); err != nil {
		return "", fmt.Errorf("error cloning the repository. error removing previous directory: %q", err)
	}

	log.Printf("Cloning the repository [%s] into [%s]\n\n", url, defaultBaseDir)
	r, err := git.PlainClone(defaultBaseDir, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		return "", fmt.Errorf("error cloning the repository: %q", err)
	}

	ref, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("error cloning the repository. error getting the HEAD reference of the repository: %q", err)
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("error cloning the repository. error getting the lattest commit of the repository: %q", err)
	}
	return commit.Hash.String(), nil
}
