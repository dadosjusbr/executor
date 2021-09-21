package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/dadosjusbr/executor"
)

func main() {
	stageGoRunEnv := map[string]string{
		"URL":           "https://raw.githubusercontent.com/dadosjusbr/coletores/master/mpal/src/output_test/membros_ativos-6-2021.json",
		"OUTPUT_FOLDER": "/output",
	}

	stagePythonRunEnv := map[string]string{
		"OUTPUT_FOLDER": "/output",
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	p := executor.Pipeline{}
	p.Name = "Tutorial"
	p.Stages = []executor.Stage{
		{
			Name:   "Get data from API Dadosjusbr",
			RunEnv: stageGoRunEnv,
			Repo:   "github.com/dadosjusbr/example-stage-go",
		},
		{
			Name: "Convert the Dadosjusbr json to csv",
			Dir:  filepath.Join(wd, "stage-python"),

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
