package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/dadosjusbr/executor"
	"github.com/dadosjusbr/executor/status"
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
	p.VolumeName = "executorTutorial"
	p.VolumeDir = "/output"
	p.Stages = []executor.Stage{
		{
			Name:   "Stage Go",
			RunEnv: stageGoRunEnv,
			Repo:   "github.com/dadosjusbr/example-stage-go",
		},
		{
			Name:    "Stage Python",
			BaseDir: wd,
			Dir:     "stage-python",
			RunEnv:  stagePythonRunEnv,
		},
	}

	result := p.Run()
	if result.Status == status.OK {
		saveReport(result, "result_pipeline_error.json")

		log.Fatalf("Error executing pipeline. Status:%v\n", result.Status)
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
