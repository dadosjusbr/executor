package executor

import (
	"fmt"
	"os"
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

func setup(dir string) error {
	if os.IsNotExist(os.Mkdir(dir+"/output", os.ModeDir)) {
		if err := os.Mkdir(dir+"/output", os.ModeDir); err != nil {
			return fmt.Errorf("error creating output folder: %q", err)
		}
	}

	cmd := "docker volume create --opt type=none --opt device=" + dir + "/output --opt o=bind --name=dadosjusbr"
	fmt.Println(cmd)
	//if err := os.exec(cmd); err != nil {
	//	return fmt.Errorf("error creating output folder: %q", err)
	//}

	return nil
}

//Run executes the pipeline
func Run(pipeline Pipeline) ([]StageExecutionResult, error) {

	if err := setup(pipeline.DefaultRepo); err != nil {
		return nil, fmt.Errorf("error in inicial setup. %q", err)

	}

	return []StageExecutionResult{}, nil
}
