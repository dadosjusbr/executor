package status

import (
	"errors"
	"fmt"
	"log"
	"os"
)

// Code is a custom type to represent ints
type Code int

const (
	// OK means that the process worked without errors.
	OK Code = 0

	// SetupError should be used for scenarios with setup problemns, like fail on create the volume for containers.
	SetupError Code = 1

	// BuildError should be used for scenarios with 'docker build' problemns.
	BuildError Code = 2

	// RunError should be used for scenarios with 'docker run' problemns.
	RunError Code = 3

	// TeardownError should be used for scenarios with teardown problemns, like fail on destroying the volume for containers.
	TeardownError Code = 4

	// InvalidParameters should be used for scenarios where month and year are not valid or mandatory parameters are empty.
	InvalidParameters Code = 5

	// SystemError should be used for scenarios like i/o erros, for example a failure on opening or writing a file.
	SystemError Code = 6

	// ConnectionError should be used for scenarios with connection problems, like timeouts or service unavailable.
	ConnectionError Code = 7

	// DataUnavailable means that the desired data was not found on crawling.
	DataUnavailable Code = 8

	//InvalidFile should be used for invalid files or for scenarios where some data could not be extracted.
	InvalidFile Code = 9

	// Unknown means that something unexpected has happend.
	Unknown Code = 10
)

var (
	statusText = map[Code]string{
		OK:                "OK",
		InvalidParameters: "Invalid Parameters",
		SystemError:       "System Error",
		ConnectionError:   "Connection Error",
		DataUnavailable:   "Data Unavailable",
		InvalidFile:       "Invalid File",
		Unknown:           "Unknown",
		SetupError:        "Setup Error",
		BuildError:        "Build Error",
		RunError:          "Run Error",
		TeardownError:     "Teardown Error",
	}
)

// Text returns a text for a status code. It returns the empty
// string if the code is unknown.
func Text(code Code) string {
	return statusText[code]
}

// ExitFromError logs the error message and call os.Exit
// passing the code if err is of type StatusError.
func ExitFromError(err error) {
	log.Println(fmt.Errorf("%q", err))
	var se *Error
	if errors.As(err, &se) {
		os.Exit(int(se.Code))
	}
	os.Exit(int(Unknown))
}
