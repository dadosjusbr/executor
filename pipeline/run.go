package pipeline

import (
	"github.com/dadosjusbr/storage"
)

//Env represents the variables used in the stages.
type Env struct {
	Month          string `envconfig:"MONTH"`
	Year           string `envconfig:"YEAR"`
	OutputFolder   string `envconfig:"OUTPUT_FOLDER"`
	MongoURI       string `envconfig:"MONGODB_URI"`
	DBName         string `envconfig:"MONGODB_DBNAME"`
	MongoMICol     string `envconfig:"MONGODB_MICOL"`
	MongoAgCol     string `envconfig:"MONGODB_AGCOL"`
	BackupArtfacts bool   `envconfig:"BACKUP_ARTFACTS"`
	SwiftUsername  string `envconfig:"SWIFT_USERNAME"`
	SwiftAPIKey    string `envconfig:"SWIFT_APIKEY"`
	SwiftAuthURL   string `envconfig:"SWIFT_AUTHURL"`
	SwiftDomain    string `envconfig:"SWIFT_DOMAIN"`
	SwiftContainer string `envconfig:"SWIFT_CONTAINER"`
}

//Stage is a phase of data release process.
type Stage struct {
	Name string
	Dir  string
}

//Pipeline represents the sequence of stages for data release.
type Pipeline struct {
	Name   string
	Path   string
	Envs   Env
	Stages []Stage
}

//Run executes the pipeline
func Run(pipeline Pipeline) ([]storage.ProcInfo, error) {

	return []storage.ProcInfo{}, nil
}
