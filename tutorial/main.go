package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/dadosjusbr/executor"
)

func main() {
	goPath := os.Getenv("GOPATH")
	repo := fmt.Sprintf("%s/src/github.com/dadosjusbr/coletores", goPath)

	cmdList := strings.Split("git rev-list -1 HEAD", " ")
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	cmd.Dir = repo
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	collectBuildEnv := map[string]string{
		"GIT_COMMIT": out.String(),
	}

	collectRunEnv := map[string]string{
		"--mes": "04",
		"--ano": "2019",
	}

	p := executor.Pipeline{}
	p.Name = "TRT13"
	p.DefaultRepo = repo
	p.Stages = []executor.Stage{
		{
			Name:     "Coleta",
			Dir:      "trt13",
			BuildEnv: collectBuildEnv,
			RunEnv:   collectRunEnv,
		},
		{
			Name: "Empacotador",
			Dir:  "trt13",
		},
	}

	_, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

}
