package executor

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dadosjusbr/executor/status"
)

// StageExecutionResult represents information about the execution of a stage.
type StageExecutionResult struct {
	Stage          Stage       `json:"stage" bson:"stage,omitempty"`                   // Name of stage.
	CommitID       string      `json:"commit" bson:"commit,omitempty"`                 // Commit of the stage repo when executing the stage.
	StartTime      time.Time   `json:"start" bson:"start,omitempty"`                   // Time at start of stage.
	FinalTime      time.Time   `json:"end" bson:"end,omitempty"`                       // Time at the end of stage.
	BuildResult    CmdResult   `json:"buildResult" bson:"buildResult,omitempty"`       // Build result.
	RunResult      CmdResult   `json:"runResult" bson:"runResult,omitempty"`           // Run result.
	SetupResult    CmdResult   `json:"setupResult" bson:"setupResult,omitempty"`       // Setup result.
	TeardownResult CmdResult   `json:"teardownResult" bson:"teardownResult,omitempty"` // Teardown result.
	Status         status.Code `json:"status" bson:"status,omitempty"`                 // Final execution status of the stage.
}

// Stage is a phase of data release process.
type Stage struct {
	Name              string            `json:"name" bson:"name,omitempt"`                                 // Stage's name.
	Dir               string            `json:"dir" bson:"dir,omitempt"`                                   // Directory to be concatenated with default base directory or with the base directory specified here in 'BaseDir'. This field is used to name the image built.
	Repo              string            `json:"repo" bson:"repo,omitempt"`                                 // Repository URL from where to clone the pipeline stage.
	BaseDir           string            `json:"base-dir" bson:"base-dir,omitempt"`                         // Base directory for the stage. This field overwrites the DefaultBaseDir in pipeline's definition.
	BuildEnv          map[string]string `json:"build-env" bson:"build-env,omitempt"`                       // Variables to be used in the stage build. They will be concatenated with the default variables defined in the pipeline, overwriting them if repeated.
	RunEnv            map[string]string `json:"run-env" bson:"run-env,omitempt"`                           // Variables to be used in the stage run. They will be concatenated with the default variables defined in the pipeline, overwriting them if repeated.
	RepoVersionEnvVar string            `json:"repo_version_env_var" bson:"repo_version_env_var,omitempt"` // Name of the environment variable passed to build and run that represents the stage commit id.
	ContainerID       string            `json:"container-id" bson:"container-id,omitempty"`                // ID of the container running used to run the stage.
	VolumeName        string            `json:"volume-name" bson:"volume-name,omitempty"`                  // Name of the shared volume.
	VolumeDir         string            `json:"volume-dir" bson:"volume-dir,omitempty"`                    // Directory of the shared volume.

	internalID string // Stage internal identification.
	index      int    // Stage position in the pipeline.
}

func (stage *Stage) run(index int, pipeline Pipeline, stdin string) (StageExecutionResult, error) {
	var ser StageExecutionResult

	stage.index = index
	stage.internalID = fmt.Sprintf("%s/%s", pipeline.Name, stage.Name)

	// 'index+1' because the index starts from 0.
	log.Printf("## Executing Stage %s [%d/%d]\n\n", stage.internalID, stage.index+1, len(pipeline.Stages))

	defer func(ser *StageExecutionResult) {
		ser.FinalTime = time.Now()
	}(&ser)

	// AQUI FIREMAN:
	// Garantir que o setup e o tear down são corretamente referenciados dentro do SER, dessa forma, eles poderão
	// Opções: campo SetupResult e TeardownResult string
	// Precisa adicionar uma variável que permita identificar se houve sucesso ou não
	ser.StartTime = time.Now()
	{
		log.Printf("### Setting up stage %s\n", stage.internalID)
		c, err := stage.setup(pipeline)
		ser.SetupResult = c
		if err != nil {
			ser.Status = status.SetupError
			log.Printf("### Error setting up stage %s:%v\n\n", stage.internalID, err)
			return ser, err
		}
		log.Printf("### Stage %s set up successfully!\n\n", stage.internalID)
	}
	ser.Stage = *stage
	{
		log.Printf("### Building stage %s\n", stage.internalID)
		c, err := stage.buildImage()
		ser.BuildResult = c
		if err != nil {
			ser.Status = status.BuildError
			log.Printf("### Error building stage %s:%v\n\n", stage.internalID, err)
			return ser, err
		}
		log.Printf("### Stage %s built successfully!\n\n", stage.internalID)
	}
	{
		log.Printf("### Running stage %s\n", stage.internalID)
		c, err := stage.runImage(stdin)
		ser.RunResult = c
		if err != nil {
			ser.Status = status.RunError
			log.Printf("### Error running stage %s:%v\n\n", stage.internalID, err)
			return ser, err
		}
		log.Printf("### Stage %s ran successfully!\n\n", stage.internalID)
	}
	{
		log.Printf("### Tearing down stage %s\n", stage.internalID)
		c, err := stage.teardown()
		ser.TeardownResult = c
		if err != nil {
			ser.Status = status.TeardownError
			log.Printf("### Error tearing down stage %s:%v\n\n", stage.internalID, err)
			return ser, err
		}
		log.Printf("### Stage %s tore down successfully!\n\n", stage.internalID)
	}
	// 'index+1' because the index starts from 0.
	log.Printf("## Stage %s [%d/%d] executed successfully!\n\n", stage.internalID, stage.index, len(pipeline.Stages))

	ser.Status = status.OK
	return ser, nil
}

func (stage *Stage) setup(pipeline Pipeline) (CmdResult, error) {
	// Even though this stage uses libraries to execute its commands, we wrap
	// those in a CmdResult to comply with the stage execution steps interface.
	if stage.BaseDir == "" {
		stage.BaseDir = pipeline.DefaultBaseDir
	}
	if stage.ContainerID == "" {
		stage.ContainerID = strings.ReplaceAll(strings.ToLower(stage.Name), " ", "-")
	}
	if stage.VolumeName == "" {
		stage.VolumeName = pipeline.VolumeName
	}
	if stage.VolumeDir == "" {
		stage.VolumeDir = pipeline.VolumeDir
	}

	// if there the field "repo" is set for the stage, clone it and update
	// its baseDir and commit id.
	if stage.Repo != "" {
		rr, err := setupRepo(stage.Repo)
		if err != nil {
			e := fmt.Errorf("error in setting up repo(%s) for stage %s setup: %w", stage.Repo, stage.Name, err)
			return CmdResult{
				Stderr:     err.Error(),
				ExitStatus: int(status.SystemError),
			}, e
		}
		// specifying commit id as environment variable.
		if stage.RepoVersionEnvVar != "" {
			if stage.BuildEnv == nil {
				stage.BuildEnv = make(map[string]string)
			}
			stage.BuildEnv[stage.RepoVersionEnvVar] = rr.commitID
			if stage.RunEnv == nil {
				stage.RunEnv = make(map[string]string)
			}
			stage.RunEnv[stage.RepoVersionEnvVar] = rr.commitID
		}
		stage.BaseDir = rr.dir
	}

	// Fill up enviroment variable maps.
	stage.BuildEnv = mergeEnv(pipeline.DefaultBuildEnv, stage.BuildEnv)
	stage.RunEnv = mergeEnv(pipeline.DefaultRunEnv, stage.RunEnv)
	return CmdResult{
		ExitStatus: int(status.OK),
	}, nil
}

func (stage *Stage) buildImage() (CmdResult, error) {
	r, err := buildImage(stage.ContainerID, stage.BaseDir, stage.Dir, stage.BuildEnv)
	if err != nil {
		return r, fmt.Errorf("error when building image: %w", err)
	}
	if status.Code(r.ExitStatus) != status.OK {
		return r, fmt.Errorf("error when building image: status code %d(%s) when building image for %s", r.ExitStatus, status.Text(status.Code(r.ExitStatus)), stage.internalID)
	}
	return r, nil
}

func (stage *Stage) runImage(stdin string) (CmdResult, error) {
	r, err := runImage(stage.ContainerID, stage.BaseDir, stage.Dir, stage.VolumeName, stage.VolumeDir, stdin, stage.RunEnv)
	if err != nil {
		return r, fmt.Errorf("error when running image: %s", err)
	}
	if status.Code(r.ExitStatus) != status.OK {
		return r, fmt.Errorf("error when running image: Status code %d(%s) when running image for %s", r.ExitStatus, status.Text(status.Code(r.ExitStatus)), stage.internalID)
	}
	return r, nil
}

// Removing temporary directories created from cloned repositories.
func (stage *Stage) teardown() (CmdResult, error) {
	// Even though this stage uses libraries to execute its commands, we wrap
	// those in a CmdResult to comply with the stage execution steps interface.
	if stage.Repo != "" {
		log.Printf("Removing cloned repo %s\n", stage.BaseDir)
		if err := os.RemoveAll(stage.BaseDir); err != nil {
			e := fmt.Errorf("error removing temp dir(%s): %w", stage.BaseDir, err)
			return CmdResult{
				Stderr:     err.Error(),
				ExitStatus: int(status.SystemError),
			}, e
		}
		log.Printf("Cloned repo temp dir removed successfully!\n")
	}
	return CmdResult{
		ExitStatus: int(status.OK),
	}, nil
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
