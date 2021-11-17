package executor

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"reflect"
	"time"

	"github.com/dadosjusbr/executor/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	noExitError   = -2
	dirPermission = 0666
)

// Pipeline represents the sequence of stages for data release.
type Pipeline struct {
	Name                 string            `json:"name" bson:"name,omitempt"`                                       // Pipeline's name.
	DefaultBaseDir       string            `json:"default-base-dir" bson:"default-base-dir,omitempt"`               // Default base directory to be used in all stages.
	DefaultBuildEnv      map[string]string `json:"default-build-env" bson:"default-build-env,omitempt"`             // Default variables to be used in the build of all stages.
	DefaultRunEnv        map[string]string `json:"default-run-env" bson:"default-run-env,omitempt"`                 // Default variables to be used in the run of all stages.
	Stages               []Stage           `json:"stages" bson:"stages,omitempt"`                                   // Confguration for the pipeline's stages.
	ErrorHandler         Stage             `json:"error-handler" bson:"error-handler,omitempt"`                     // Default stage to deal with any errors that occur in the execution of the pipeline.
	SkipVolumeDirCleanup bool              `json:"skip-volume-dir-cleanup" bson:"skip-volume-dir-cleanup,omitempt"` // Skip pipeline's volume setup. Useful for debugging long-running pipelines.
	VolumeDir            string            `json:"volume-dir" bson:"volume-dir,omitempt"`                           // Pipeline's output directory. Shared accross all pipeline stages.
	VolumeName           string            `json:"volume-name" bson:"volume-name,omitempt"`                         // Pipeline's name. Shared accross all pipeline stages.
}

// PipelineResult represents the pipeline information and their results.
type PipelineResult struct {
	Name           string                 `json:"name" bson:"name,omitempty"`               // Name of pipeline.
	StageResults   []StageExecutionResult `json:"stageResult" bson:"stageResult,omitempty"` // Results of stage execution.
	SetupResult    string
	TeardownResult string
	StartTime      time.Time   `json:"start" bson:"start,omitempty"`   // Time at start of pipeline.
	FinalTime      time.Time   `json:"final" bson:"final,omitempty"`   // Time at end of pipeline.
	Status         status.Code `json:"sucess" bson:"status,omitempty"` // Whether the pipeline was successfull.
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
func (p *Pipeline) Run() PipelineResult {
	result := PipelineResult{Name: p.Name, StartTime: time.Now()}

	defer func(ser *PipelineResult) {
		result.FinalTime = time.Now()
	}(&result)

	log.Println()
	log.Printf("# Setting up Pipeline %s\n", p.Name)
	if err := p.setup(); err != nil {
		result.SetupResult = fmt.Sprintf("Error in setup: %q", err)
		result.Status = status.SetupError
		log.Printf("# Error setting up pipeline %s:%v\n\n", p.Name, err)
		return result
	}
	log.Printf("# Pipeline %s set up successfully!\n\n", p.Name)

	for index, stage := range p.Stages {
		fmt.Printf("\n")
		stdin := ""
		if index == 0 {
			// https://stackoverflow.com/a/38612652
			// check if stdin has data and if it comes from a pipe.
			fi, err := os.Stdin.Stat()
			if err != nil {
				log.Printf("Error verifying stdin: %q. Proceeding...\n", err)
			}
			// only consumes data if it comes from a pipe.
			if fi.Mode()&fs.ModeCharDevice == 0 {
				in, err := io.ReadAll(os.Stdin)
				if err != nil {
					log.Printf("Error reading data from stdin: %q. Proceeding...\n", err)
				} else {
					stdin = string(in)
				}
			}
		} else {
			// 'index-1' is accessing the output from previous stage.
			stdin = result.StageResults[index-1].RunResult.Stdout
		}

		// TODO: Move tearing down to the stage.
		ser, err := stage.run(index, *p, stdin)
		result.StageResults = append(result.StageResults, ser)
		if err != nil {
			// We don't want teardown the stage twice.
			if ser.Status != status.TeardownError {
				log.Printf("### Tearing down stage %s\n", stage.internalID)
				if _, err := stage.teardown(); err != nil {
					log.Printf("### Error tearing down stage %s:%v\n\n", stage.internalID, err)
				} else {
					log.Printf("### Stage %s tore down successfully!\n\n", stage.internalID)
				}
			}

			// If the error handler stage fails, the pipeline simply logs and proceeed.
			her, err := p.handleError(ser, result)
			if err != nil && !reflect.ValueOf(p.ErrorHandler).IsZero() && her.Status != status.TeardownError {
				log.Printf("### Tearing down stage %s\n", p.ErrorHandler.internalID)
				if _, err := p.ErrorHandler.teardown(); err != nil {
					log.Printf("### Error tearing down stage %s:%v\n\n", p.ErrorHandler.internalID, err)
				} else {
					log.Printf("### Stage %s tore down successfully!\n\n", p.ErrorHandler.internalID)
				}
			}
			result.StageResults = append(result.StageResults, her)
		}
		result.Status = ser.Status

		// If there is an error, stop the pipeline
		if ser.Status != status.OK {
			break
		}
	}

	log.Printf("# Tearing down pipeline %s\n", p.Name)
	if err := p.teardown(); err != nil {
		result.Status = status.TeardownError
		result.TeardownResult = fmt.Sprintf("Error in teardown: %q", err)
		log.Printf("# Error tearing down pipeline %s:%v\n\n", p.Name, err)
		return result
	}
	log.Printf("# Pipeline %s tore down successfully!\n\n", p.Name)
	return result
}

func (p *Pipeline) setup() error {
	log.Printf("Checking pipeline spec validation\n")
	for _, s := range p.Stages {
		if err := s.validateSpec(); err != nil {
			return fmt.Errorf("Stage %s spec validation failed:%v", s.Name, err)
		}
	}
	log.Printf("Spec validated successfully!\n")

	if p.VolumeDir == "" || p.VolumeName == "" {
		log.Printf("volume-dir or volume-name not set, skipping shared volume setup.")
		return nil
	}

	log.Printf("Setting up directory:%s\n", p.VolumeDir)
	log.Printf("$ mkdir -m %d %s", dirPermission, p.VolumeDir)
	if err := os.MkdirAll(p.VolumeDir, dirPermission); err != nil {
		return fmt.Errorf("error (re)creating shared dir(%s) with permissions(%d): %w", p.VolumeDir, dirPermission, err)
	}
	log.Printf("Directory %s created sucessfully!\n", p.VolumeDir)

	log.Printf("Creating volume %s:%s\n", p.VolumeName, p.VolumeDir)
	if err := createVolume(p.VolumeDir, p.VolumeName); err != nil {
		return err
	}
	log.Printf("Volume %s:%s create sucessfully!\n", p.VolumeName, p.VolumeDir)
	return nil
}

func (p *Pipeline) teardown() error {
	if p.VolumeDir == "" || p.VolumeName == "" {
		log.Printf("volume-dir or volume-name not set, skipping shared volume teardown.")
		return nil
	}

	log.Printf("Removing volume %s:%s\n", p.VolumeName, p.VolumeDir)
	if err := removeVolume(p.VolumeName); err != nil {
		return err
	}
	log.Printf("Volume %s:%s removed sucessfully!\n", p.VolumeName, p.VolumeDir)

	if p.SkipVolumeDirCleanup {
		log.Printf("Skipping removing volume directory")
		return nil
	}
	log.Printf("Removing directory:%s\n", p.VolumeDir)
	log.Printf("$ rm -rf %s", p.VolumeDir)
	if err := os.RemoveAll(p.VolumeDir); err != nil {
		return fmt.Errorf("error removing volume dir(%s): %q", p.VolumeDir, err)
	}
	log.Printf("Directory %s removed sucessfully!\n", p.VolumeDir)
	return nil
}

func (p *Pipeline) handleError(ser StageExecutionResult, result PipelineResult) (StageExecutionResult, error) {
	handler := p.ErrorHandler
	// TODO(danielfireman): make the whole pipeline use this proto
	pDef := PipelineDef{
		Name:                 p.Name,
		DefaultBaseDir:       p.DefaultBaseDir,
		DefaultBuildEnv:      p.DefaultBuildEnv,
		DefaultRunEnv:        p.DefaultRunEnv,
		VolumeDir:            p.VolumeDir,
		SkipVolumeDirCleanup: p.SkipVolumeDirCleanup,
		ErrorHander:          stage2stageDef(p.ErrorHandler),
	}
	for _, s := range p.Stages {
		pDef.Stages = append(pDef.Stages, stage2stageDef(s))
	}
	pExec := PipelineExecution{
		Pipeline:         &pDef,
		SetupErrorMsg:    result.SetupResult,
		TeardownErrorMsg: result.TeardownResult,
	}
	for _, s := range result.StageResults {
		pExec.Results = append(pExec.Results, &StageExecution{
			StartTime:   timestamppb.New(s.StartTime),
			FinishTime:  timestamppb.New(s.FinalTime),
			ContainerId: s.Stage.ContainerID,
			CommitId:    s.CommitID,
			Setup:       cmdResult2StepExec(s.BuildResult),
			Build:       cmdResult2StepExec(s.BuildResult),
			Run:         cmdResult2StepExec(s.RunResult),
			Teardown:    cmdResult2StepExec(s.TeardownResult),
			Status:      StageExecution_Status(s.Status),
		})
	}
	stdin, err := prototext.Marshal(&pExec)
	if err != nil {
		log.Printf("### Error marshaling execution result for default error handling:%s. Skipping default error handling.\n\n", string(stdin))
		return StageExecutionResult{}, err
	}

	// NOTE: reflect about making the default error handler: should it become a normal stage?
	if reflect.ValueOf(handler).IsZero() {
		log.Printf("### Executing default error handling. Printing information about last stage execution:\n\n")
		log.Printf("%s\n", string(stdin))
		log.Printf("### Default error handling stage executed successfully!\n\n")
		return StageExecutionResult{Status: status.OK}, nil
	}
	return handler.run(-1, *p, string(stdin))
}

func cmdResult2StepExec(r CmdResult) *StepExecution {
	return &StepExecution{
		Stdin:      r.Stdin,
		Stdout:     r.Stdout,
		Stderr:     r.Stderr,
		Cmd:        r.Cmd,
		CmdDir:     r.Cmd,
		StatusCode: int32(r.ExitStatus),
		Env:        r.Env,
		StartTime:  timestamppb.New(r.StartTime),
		FinishTime: timestamppb.New(r.FinishTime),
	}
}

func stage2stageDef(s Stage) *StageDef {
	return &StageDef{
		Name:              s.Name,
		Dir:               s.Dir,
		BaseDir:           s.BaseDir,
		Repo:              s.Repo,
		RepoVersionEnvVar: s.RepoVersionEnvVar,
		BuildEnv:          s.BuildEnv,
		RunEnv:            s.RunEnv,
	}
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
// func handleError(result *PipelineResult, previousSer StageExecutionResult, previousStatus status.Code, msg string, handler Stage) (PipelineResult, error) {
// 	if reflect.ValueOf(handler).IsZero() {
// 		log.Printf("Error happened. Starting default handling routine.")
// 		erStdin, err := json.MarshalIndent(previousSer, "", "\t")
// 		if err != nil {
// 			return *result, status.NewError(status.ErrorHandlerError, fmt.Errorf("error marshaling input of error handler: %s", err))
// 		}
// 		fmt.Println(string(erStdin))
// 		log.Printf("Pipeline finished with error. Please cehck the logs for further details.")
// 		return *result, fmt.Errorf(msg)
// 	}
// 	previousSer.FinalTime = time.Now()
// 	result.StageResults = append(result.StageResults, previousSer)

// 	if handler.Repo != "" {
// 		if err := handler.setup(); err != nil {
// 			return *result, status.NewError(status.SetupError, fmt.Errorf("error when setting up image for error handler: %s", err))
// 		}
// 	}

// 	if handler.BaseDir != "" {
// 		var serError StageExecutionResult
// 		var err error
// 		serError.Stage = handler
// 		serError.StartTime = time.Now()

// 		id := fmt.Sprintf("%s/%s calls Error Handler", result.Name, previousSer.Stage.Name)
// 		serError.BuildResult, err = buildImage(id, handler.BaseDir, handler.Dir, handler.BuildEnv)
// 		if err != nil {
// 			result.StageResults = append(result.StageResults, serError)
// 			result.Status = status.Text(status.ErrorHandlerError)
// 			result.FinalTime = time.Now()

// 			return *result, status.NewError(status.BuildError, fmt.Errorf("error when building image for error handler: %s", err))
// 		}
// 		if status.Code(serError.BuildResult.ExitStatus) != status.OK {
// 			result.StageResults = append(result.StageResults, serError)
// 			result.Status = status.Text(status.ErrorHandlerError)
// 			result.FinalTime = time.Now()

// 			return *result, status.NewError(status.BuildError, fmt.Errorf("error when building image for error handler: Status code %d(%s) when running image for %s", serError.BuildResult.ExitStatus, status.Text(status.Code(serError.BuildResult.ExitStatus)), id))
// 		}

// 		erStdin, err := json.Marshal(previousSer)
// 		if err != nil {
// 			return *result, status.NewError(status.ErrorHandlerError, fmt.Errorf("error in parser StageExecutionResult for error handler: %s", err))
// 		}
// 		serError.RunResult, err = runImage(id, handler.BaseDir, handler.Dir, string(erStdin), handler.RunEnv)
// 		if err != nil {
// 			result.StageResults = append(result.StageResults, serError)
// 			result.Status = status.Text(status.ErrorHandlerError)
// 			result.FinalTime = time.Now()

// 			return *result, status.NewError(status.RunError, fmt.Errorf("error when running image for error handler: %s", err))
// 		}
// 		if status.Code(serError.RunResult.ExitStatus) != status.OK {
// 			result.StageResults = append(result.StageResults, serError)
// 			result.Status = status.Text(status.ErrorHandlerError)
// 			result.FinalTime = time.Now()

// 			return *result, status.NewError(status.RunError, fmt.Errorf("error when running image for error handler: Status code %d(%s) when running image for %s", serError.RunResult.ExitStatus, status.Text(status.Code(serError.RunResult.ExitStatus)), id))
// 		}
// 	}

// 	result.Status = status.Text(previousStatus)
// 	result.FinalTime = time.Now()
// 	return *result, fmt.Errorf(msg)
// }

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
