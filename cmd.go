package executor

import "time"

// CmdResult represents information about a execution of a command.
type CmdResult struct {
	Stdin      string    `json:"stdin" bson:"stdin,omitempt"`              // String containing the standard input of the process.
	Stdout     string    `json:"stdout" bson:"stdout,omitempty"`           // String containing the standard output of the process.
	Stderr     string    `json:"stderr" bson:"stderr,omitempty"`           // String containing the standard error of the process.
	Cmd        string    `json:"cmd" bson:"cmd,omitempty"`                 // Command that has been executed.
	CmdDir     string    `json:"cmdDir" bson:"cmdir,omitempty"`            // Local directory, in which the command has been executed.
	ExitStatus int       `json:"status" bson:"status,omitempty"`           // Exit code of the process executed.
	Env        []string  `json:"env" bson:"env,omitempty"`                 // Copy of strings representing the environment variables in the form ke=value.
	StartTime  time.Time `json:"start_time" bson:"start_time,omitempty"`   // Timestamp the command execution starts.
	FinishTime time.Time `json:"finish_time" bson:"finish_time,omitempty"` // Timestamp the command execution finishes.
}
