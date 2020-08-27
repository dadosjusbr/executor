# Tutorial de utilização do pacote *executor*

*[O tutorial também pode ser visualizado no medium]().

Vamos montar um pipeline com 2 estágios e 2 programas escritos em linguagens diferentes.

- [Definindo o primeiro estágio]()
- [Definindo o segundo estágio]()
- [Definindo o Pipeline]()


## Primeiro estágio: [stage-go](https://github.com/dadosjusbr/executor/tree/master/tutorial/stage-go)
O primeiro programa é escrito em Go. Ele faz uma requisição na API do DadosJusBR, salva o arquivo em formato json e imprime o resultado na saída padrão.

### [*Dockerfile*](https://github.com/dadosjusbr/executor/blob/master/tutorial/stage-go/Dockerfile)
``` dockerfile
FROM golang:1.14.0-alpine

# Create output folder
RUN mkdir /output

# Move to working directory /app
WORKDIR /app

# Copy the code into the container
COPY . .

# Set necessary environmet variables needed
ENV GO111MODULE=on 

# Install package status
RUN apk update && apk add git

# Build the application
RUN go build -o main

ENTRYPOINT ["./main"]
```
Repare na criação da pasta `/output` dentro do container. É essa pasta que será espelhada com a nossa pasta `tutorial/output`local.

### [*main.go*](https://github.com/dadosjusbr/executor/blob/master/tutorial/stage-go/main.go)
``` go
package main

import (
	"github.com/dadosjusbr/executor/status"
)

func main() {
	url := os.Getenv("URL")
	if url == "" {
		status.ExitFromError(status.NewError(status.InvalidParameters, fmt.Errorf("URL env var can not be empty")))
	}
	output := os.Getenv("OUTPUT_FOLDER")
	if output == "" {
		status.ExitFromError(status.NewError(status.InvalidParameters, fmt.Errorf("OUTPUT_FOLDER env var can not be empty")))
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(status.DataUnavailable)
		status.ExitFromError(status.NewError(status.DataUnavailable, fmt.Errorf("error requesting url: %s", err)))
	}
	if resp.StatusCode != 200 {
		log.Fatalf("http status is not 200 OK. request returned the %d status", resp.StatusCode)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		status.ExitFromError(status.NewError(status.DataUnavailable, fmt.Errorf("error reading data: %s", err)))
	}

	pathFile := fmt.Sprintf("%s/result.json", output)
	err = ioutil.WriteFile(pathFile, data, 0666)
	if err != nil {
		status.ExitFromError(status.NewError(status.SystemError, fmt.Errorf("error writing file: %s", err)))
	}

	fmt.Println(string(data))
}
```
Um ponto a ser destacado aqui é a utilização do pacote [status](https://github.com/dadosjusbr/executor/tree/master/status). Esse pacote tem o objetivo de padronizar os status de execução dos coletores DadosJusBR e de possíveis erros em estágios de um Pipeline.

## Segundo estágio: [stage-python](https://github.com/dadosjusbr/executor/tree/master/tutorial/stage-python)
O segundo programa consome dados da entrada padrão e, usando a biblioteca Pandas na linguagem Python, transforma os dados aninhados (no formato json), para um formato tabular. Por fim, salva os dados em um arquivo .csv.

### [*Dockerfile*](https://github.com/dadosjusbr/executor/blob/master/tutorial/stage-python/Dockerfile)
``` dockerfile
FROM python:3.7.2-slim

# Create output folder
RUN mkdir /output

# Move to working directory /app
WORKDIR /app

# Copy the code into the container
COPY . .

# Installing dependencies
RUN pip install --upgrade pip
RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "./script.py"]
```
Repare na criação da pasta `/output` dentro do container. É essa pasta que será espelhada com a nossa pasta `tutorial/output`local.

### [*script.py*](https://github.com/dadosjusbr/executor/blob/master/tutorial/stage-python/script.py)
``` python
import sys
import pandas as pd
import json
import os


data = sys.stdin.read()  
data = json.loads(data)

df = pd.json_normalize(data)

output = os.environ['OUTPUT_FOLDER']
file_name = '{}/result.csv'.format(output)
df.to_csv(file_name, index=False)
```

## Montagem do Pipeline

### [*tutorial.go*](https://github.com/dadosjusbr/executor/blob/master/tutorial/tutorial.go)
``` go
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
```

### Resultado do Pipeline

Após a execução, nossa pasta tutorial fica da seguinte forma:
```
tutorial
|
|__ output/
| |__ result.json
| |__ result.csv
|
|__ stage-go/
| |__ Dockerfile
| |__ main.go
|
|__ stage-python/
| |__ Dockerfile
| |__ script.
|
|__ tutorial.go
|__ result_pipeline.json
```

E esse é o nosso [result_pipeline.json](https://gist.github.com/Lorenaps/8392742a733344f7001f70eaa4b05e72):

``` json
{
 "name": "Tutorial",
 "stageResult": [
  {
   "stage": "Get data from API Dadosjusbr",
   "start": "2020-08-21T14:47:59.536805914-03:00",
   "end": "2020-08-21T14:48:06.906129997-03:00",
   "buildResult": {
    "stdin": "",
    "stdout": "Sending build context to Docker daemon  6.144kB\r\r\nStep 1/8 : FROM golang:1.14.0-alpine\n ---\u003e 51e47ee4db58\nStep 2/8 : RUN mkdir /output\n ---\u003e Using cache\n ---\u003e 6d2a642ff06b\nStep 3/8 : WORKDIR /app\n ---\u003e Using cache\n ---\u003e a9ecbdbb96a6\nStep 4/8 : COPY . .\n ---\u003e Using cache\n ---\u003e 8d87932eaae6\nStep 5/8 : ENV GO111MODULE=on\n ---\u003e Using cache\n ---\u003e 568175c6a7fd\nStep 6/8 : RUN apk update \u0026\u0026 apk add git\n ---\u003e Using cache\n ---\u003e 5278823cd5b8\nStep 7/8 : RUN go build -o main\n ---\u003e Using cache\n ---\u003e 5d10280b6f3a\nStep 8/8 : ENTRYPOINT [\"./main\"]\n ---\u003e Using cache\n ---\u003e 07a5f226c304\nSuccessfully built 07a5f226c304\nSuccessfully tagged stage-go:latest\n",
    "stderr": "",
    "cmd": "docker build -t stage-go .",
    "cmdDir": "/home/lsp/projetos/go/src/github.com/dadosjusbr/executor/tutorial/stage-go",
    "status": 0,
    "env": ""
   },
   "runResult": {
    "stdin": "",
    "stdout": "[\n {\n  \"reg\": \"29949\",\n  \"name\": \"ADRIANA LEMES FERNANDES MARACAJA COUTINHO\",\n  \"role\": \"JUIZ SUBSTITUTO - NÍVEL SUPERIOR JSJS\",\n  \"type\": \"membro\",\n  \"workplace\": \"6ª VARA DO TRABALHO DE CAMPINA GRANDE\",\n  \"active\": true,\n  \"income\": {\n   \"total\": 34599.19,\n   \"wage\": 32004.65,\n   \"perks\": {\n    \"total\": 910.08,\n    \"food\": null,\n    \"transportation\": null,\n    \"pre_school\": null,\n    \"health\": null,\n    \"birth_aid\": null,\n    \"housing_aid\": null,\n    \"subsistence\": null,\n    \"others\": null\n   },\n   \"other\": {\n    \"total\": 1684.46,\n    \"person_benefits\": 0,\n    \"eventual_benefits\": 1684.46,\n    \"trust_position\": null,\n    \"daily\": 0,\n    \"gratification\": 0,\n    \"origin_pos\": 0,\n    \"others\": null\n   }\n  },\n  \"discounts\": {\n   \"total\": -10612.25,\n   \"prev_contribution\": -3058.08,\n   \"ceil_retention\": 0,\n   \"income_tax\": -7554.17,\n   \"other\": {\n    \"other_discounts\": 0\n   }\n  }\n }\n]\n\n",
    "stderr": "",
    "cmd": "docker run -i -v dadosjusbr:/output --rm --env URL=https://dadosjusbr.org/api/v1/orgao/trt13/2020/4 --env OUTPUT_FOLDER=/output stage-go",
    "cmdDir": "/home/lsp/projetos/go/src/github.com/dadosjusbr/executor/tutorial/stage-go",
    "status": 0,
    "env": ""
   }
  },
  {
   "stage": "Convert the Dadosjusbr json to csv",
   "start": "2020-08-21T14:48:06.90613194-03:00",
   "end": "2020-08-21T14:52:04.215821387-03:00",
   "buildResult": {
    "stdin": "",
    "stdout": "Sending build context to Docker daemon  4.096kB\r\r\nStep 1/7 : FROM python:3.7.2-slim\n ---\u003e f46a51a4d255\nStep 2/7 : RUN mkdir /output\n ---\u003e Running in f1055c9905d3\nRemoving intermediate container f1055c9905d3\n ---\u003e ec07becad9cc\nStep 3/7 : WORKDIR /app\n ---\u003e Running in 708b232a6499\nRemoving intermediate container 708b232a6499\n ---\u003e 49fb2d6368b0\nStep 4/7 : COPY . .\n ---\u003e 7183f7f74834\nStep 5/7 : RUN pip install --upgrade pip\n ---\u003e Running in eb0d51f3969d\nCollecting pip\n  Downloading https://files.pythonhosted.org/packages/5a/4a/39400ff9b36e719bdf8f31c99fe1fa7842a42fa77432e584f707a5080063/pip-20.2.2-py2.py3-none-any.whl (1.5MB)\nInstalling collected packages: pip\n  Found existing installation: pip 19.0.3\n    Uninstalling pip-19.0.3:\n      Successfully uninstalled pip-19.0.3\nSuccessfully installed pip-20.2.2\nRemoving intermediate container eb0d51f3969d\n ---\u003e bab49b3c2c44\nStep 6/7 : RUN pip install --no-cache-dir -r requirements.txt\n ---\u003e Running in 901ce120b456\nCollecting numpy==1.19.0\n  Downloading numpy-1.19.0-cp37-cp37m-manylinux2010_x86_64.whl (14.6 MB)\nCollecting pandas==1.0.5\n  Downloading pandas-1.0.5-cp37-cp37m-manylinux1_x86_64.whl (10.1 MB)\nCollecting python-dateutil==2.8.1\n  Downloading python_dateutil-2.8.1-py2.py3-none-any.whl (227 kB)\nCollecting pytz==2020.1\n  Downloading pytz-2020.1-py2.py3-none-any.whl (510 kB)\nCollecting six==1.15.0\n  Downloading six-1.15.0-py2.py3-none-any.whl (10 kB)\nInstalling collected packages: numpy, pytz, six, python-dateutil, pandas\nSuccessfully installed numpy-1.19.0 pandas-1.0.5 python-dateutil-2.8.1 pytz-2020.1 six-1.15.0\nRemoving intermediate container 901ce120b456\n ---\u003e fb77bc3632f9\nStep 7/7 : CMD [\"python\", \"./script.py\"]\n ---\u003e Running in 7e2ed0033146\nRemoving intermediate container 7e2ed0033146\n ---\u003e 5483ee8fc5e6\nSuccessfully built 5483ee8fc5e6\nSuccessfully tagged stage-python:latest\n",
    "stderr": "",
    "cmd": "docker build -t stage-python .",
    "cmdDir": "/home/lsp/projetos/go/src/github.com/dadosjusbr/executor/tutorial/stage-python",
    "status": 0,
    "env": ""
   },
   "runResult": {
    "stdin": "[\n {\n  \"reg\": \"29949\",\n  \"name\": \"ADRIANA LEMES FERNANDES MARACAJA COUTINHO\",\n  \"role\": \"JUIZ SUBSTITUTO - NÍVEL SUPERIOR JSJS\",\n  \"type\": \"membro\",\n  \"workplace\": \"6ª VARA DO TRABALHO DE CAMPINA GRANDE\",\n  \"active\": true,\n  \"income\": {\n   \"total\": 34599.19,\n   \"wage\": 32004.65,\n   \"perks\": {\n    \"total\": 910.08,\n    \"food\": null,\n    \"transportation\": null,\n    \"pre_school\": null,\n    \"health\": null,\n    \"birth_aid\": null,\n    \"housing_aid\": null,\n    \"subsistence\": null,\n    \"others\": null\n   },\n   \"other\": {\n    \"total\": 1684.46,\n    \"person_benefits\": 0,\n    \"eventual_benefits\": 1684.46,\n    \"trust_position\": null,\n    \"daily\": 0,\n    \"gratification\": 0,\n    \"origin_pos\": 0,\n    \"others\": null\n   }\n  },\n  \"discounts\": {\n   \"total\": -10612.25,\n   \"prev_contribution\": -3058.08,\n   \"ceil_retention\": 0,\n   \"income_tax\": -7554.17,\n   \"other\": {\n    \"other_discounts\": 0\n   }\n  }\n }\n]\n\n",
    "stdout": "",
    "stderr": "",
    "cmd": "docker run -i -v dadosjusbr:/output --rm --env OUTPUT_FOLDER=/output stage-python",
    "cmdDir": "/home/lsp/projetos/go/src/github.com/dadosjusbr/executor/tutorial/stage-python",
    "status": 0,
    "env": ""
   }
  }
 ],
 "start": "2020-08-21T14:47:59.450664564-03:00",
 "final": "2020-08-21T14:52:04.361930031-03:00",
 "status": 0
}
```
