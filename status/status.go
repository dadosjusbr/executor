package status

// Code is a custom type to represent ints
type Code int

const (
	// OK means that the process worked without errors
	OK Code = 0

	// InvalidParameters should be used for scenarios where month and year are not valid
	InvalidParameters Code = 1

	// SystemError should be used for scenarios like i/o erros, for example a failure on opening a file
	SystemError Code = 2

	// ConnectionError should be used for scenarios with connection problems, like timeouts or service unavailable
	ConnectionError Code = 3

	// DataUnavailable means that the desired data was not found on crawling
	DataUnavailable Code = 4

	//InvalidFile should be used for invalid files or for scenarios where some data could not be extracted
	InvalidFile Code = 5

	// Unknown means that something unexpected has happend
	Unknown Code = 6

	// SetupError should be used for scenarios with setup problemns, like fail on create the volume for containers.
	SetupError Code = 7

	// BuildError should be used for scenarios with 'docker build' problemns.
	BuildError Code = 8

	// RunError should be used for scenarios with 'docker run' problemns.
	RunError Code = 9

	// ErrorHandlerError should be used for scenarios where the HandlerError stage return error.
	ErrorHandlerError Code = 10
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
	}
)

// Text returns a text for a status code. It returns the empty
// string if the code is unknown.
func Text(code Code) string {
	return statusText[code]
}
