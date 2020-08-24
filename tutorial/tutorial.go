package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dadosjusbr/executor"
)

func main() {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		log.Fatal("GOPATH env var can not be empty")
	}
	baseDir := fmt.Sprintf("%s/src/github.com/dadosjusbr/executor/tutorial", goPath)

	stageGoRunEnv := map[string]string{
		"URL":           "https://dadosjusbr.org/api/v1/orgao/trt13/2020/4",
		"OUTPUT_FOLDER": "/output",
	}

	stagePythonRunEnv := map[string]string{
		"OUTPUT_FOLDER": "/output",
	}

	p := executor.Pipeline{}
	p.Name = "Tutorial"
	p.DefaultBaseDir = baseDir
	p.Stages = []executor.Stage{
		{
			Name:   "Get data from API Dadosjusbr",
			Dir:    "stage-go",
			RunEnv: stageGoRunEnv,
		},
		{
			Name:   "Convert the Dadosjusbr json to csv",
			Dir:    "stage-python",
			RunEnv: stagePythonRunEnv,
		},
	}

	result, err := p.Run()
	if err != nil {
		saveReport(result, "result_pipeline_error.json")

		log.Fatal(err)
	}
	saveReport(result, "result_pipeline.json")

}

func saveReport(result executor.PipelineResult, fileName string) {
	resultJSON, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(fileName, resultJSON, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
