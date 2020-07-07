package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dadosjusbr/executor"
)

func main() {
	goPath := os.Getenv("GOPATH")
	collectBuildEnv := map[string]string{
		"GIT_COMMIT": "9bdeec22238644fa78ff7e3e9ab6f126fcaefd29",
	}

	p := executor.Pipeline{}
	p.Name = "mppb"
	p.DefaultRepo = fmt.Sprintf("%s/src/github.com/dadosjusbr/coletores", goPath)
	p.Stages = []executor.Stage{
		{
			Name:     "Coleta",
			Dir:      "mppb",
			BuildEnv: collectBuildEnv,
		},
	}

	_, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

}
