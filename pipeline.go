package executor

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

//Run executes the pipeline
func Run(pipeline Pipeline) ([]StageExecutionResult, error) {

	return []StageExecutionResult{}, nil
}
