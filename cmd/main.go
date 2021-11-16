package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/dadosjusbr/executor"
	"github.com/dadosjusbr/executor/status"
	"github.com/spf13/pflag"
)

var (
	input          = pflag.String("in", "", "Path for the descriptor file.")
	volumeName     = pflag.String("volume-name", "", "Shared volume name.")
	volumeDir      = pflag.String("volume-dir", "", "Shared volume full path.")
	defaultBaseDir = pflag.String("def-base-dir", "", "Base path to search for stages and to place the cloned repositorie")
	defaultEnvFlag = pflag.StringSlice("def-run-env", []string{}, "Environment variables that override the default vars.")
)

func main() {
	pflag.Parse()

	defaultEnv := make(map[string]string)
	for _, e := range *defaultEnvFlag {
		env := strings.Split(e, ":")
		if len(env) != 2 {
			log.Fatalf("Invalid env var spec: %s", e)
		}
		defaultEnv[env[0]] = env[1]
	}

	if *input == "" {
		log.Fatal("Path to the input file not found. Forgot --in?")
	}

	in, err := ioutil.ReadFile(*input)
	if err != nil {
		log.Fatalf("Erro lendo dados da entrada padrão: %q", err)
	}

	var p executor.Pipeline
	if err := json.Unmarshal(in, &p); err != nil {
		log.Fatalf("Erro convertendo pipeline da entrada padrão: %q\n\"%s\"", err, string(in))
	}

	p.DefaultRunEnv = mergeMaps(p.DefaultRunEnv, defaultEnv) // merging maps.
	log.Printf("Pipeline: %+v\n\n", p)

	// the flag replaces the pipeline description. Useful at runtime.
	if *volumeName != "" {
		p.VolumeName = *volumeName
	}
	if p.VolumeName == "" {
		log.Printf("Você não setou o campo volume-name, usando \"dadosjusbr\"")
		p.VolumeName = "dadosjusbr"
	}

	if *volumeDir != "" {
		p.VolumeDir = *volumeDir
	}
	if p.VolumeDir == "" {
		log.Printf("Você não setou o campo volume-name, usando \"dadosjusbr\"")
		p.VolumeDir = "/output"
	}

	if *defaultBaseDir != "" {
		p.DefaultBaseDir = *defaultBaseDir
	}

	log.Printf("Executando pipeline %s", p.Name)
	result := p.Run()
	if result.Status != status.OK {
		log.Printf("Erro executando pipeline: %s. Imprimindo resultado:\n\n", p.Name)
		log.Printf("%+v", result)
		return
	}
	log.Printf("Pipeline %s executado com sucesso! Imprimindo resultado:\n\n", p.Name)
	fmt.Printf("%+v", result)
}

// mergeMaps adds all elements of sec to first.
func mergeMaps(first, sec map[string]string) map[string]string {
	if first == nil {
		return sec
	}
	env := make(map[string]string)
	for k, v := range first {
		env[k] = v
	}
	for k, v := range sec {
		env[k] = v
	}
	return env
}
