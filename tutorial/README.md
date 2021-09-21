# Tutorial

*[O tutorial também pode ser visualizado no medium]().

Vamos montar um pipeline com 2 estágios e 2 programas escritos em linguagens diferentes.

- [Definindo o primeiro estágio](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#primeiro-est%C3%A1gio-stage-go)
	- [Dockerfile](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#dockerfile)
	- [main.go](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#maingo)
- [Definindo o segundo estágio](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#segundo-est%C3%A1gio-stage-python)
	- [Dockerfile](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#dockerfile-1)
	- [script.py](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#scriptpy)
- [Definindo o Pipeline](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#montagem-do-pipeline)
	- [tutorial.go](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#tutorialgo)
- [Resultado do Pipeline](https://github.com/dadosjusbr/executor/blob/master/tutorial/README.md#resultado-do-pipeline)


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

### [*main.go*](https://github.com/dadosjusbr/example-stage-go/blob/master/main.go)
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
Notem que o primeiro estágio está em outro repositório. Este repositório vai ser clonado em um diretório temporário e então a compilação e execução do contêiner ocorrerão.

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
			Repo:   "https://github.com/dadosjusbr/example-stage-go",
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
|__ stage-python/
| |__ Dockerfile
| |__ script.
|
|__ tutorial.go
|__ result_pipeline.json
```

E esse é o nosso [result_pipeline.json](https://gist.github.com/danielfireman/e742fdcc4f21b4592bfb70b14d172248):

``` json
{
 "name": "Tutorial",
 "stageResult": [
  {
   "stage": {
    "name": "Get data from API Dadosjusbr",
    "dir": "",
    "repo": "https://github.com/dadosjusbr/example-stage-go",
    "base-dir": "",
    "build-env": null,
    "run-env": {
     "OUTPUT_FOLDER": "/output",
     "URL": "https://raw.githubusercontent.com/dadosjusbr/coletores/master/mpal/src/output_test/membros_ativos-6-2021.json"
    }
   },
   "commit": "2c37f8b833b869029c409a731ff64674b21d467d",
   "start": "2021-09-20T21:27:47.386103633-03:00",
   "end": "2021-09-20T21:27:58.945773405-03:00",
   "buildResult": {
    "stdin": "",
    "stdout": "Sending build context to Docker daemon  19.97kB\r\r\nStep 1/8 : FROM golang:1.14.0-alpine\n ---\u003e 51e47ee4db58\nStep 2/8 : RUN mkdir /output\n ---\u003e Using cache\n ---\u003e bd0824b6a60e\nStep 3/8 : WORKDIR /app\n ---\u003e Using cache\n ---\u003e 33e7b235c63f\nStep 4/8 : COPY . .\n ---\u003e b00f5ecb5cba\nStep 5/8 : ENV GO111MODULE=on\n ---\u003e Running in da3a396fca1c\nRemoving intermediate container da3a396fca1c\n ---\u003e bbad03f3d0ee\nStep 6/8 : RUN apk update \u0026\u0026 apk add git\n ---\u003e Running in aa4b376f1b0c\nfetch http://dl-cdn.alpinelinux.org/alpine/v3.11/main/x86_64/APKINDEX.tar.gz\nfetch http://dl-cdn.alpinelinux.org/alpine/v3.11/community/x86_64/APKINDEX.tar.gz\nv3.11.12-14-g49b29cee4b [http://dl-cdn.alpinelinux.org/alpine/v3.11/main]\nv3.11.11-124-gf2729ece5a [http://dl-cdn.alpinelinux.org/alpine/v3.11/community]\nOK: 11284 distinct packages available\n(1/5) Installing nghttp2-libs (1.40.0-r1)\n(2/5) Installing libcurl (7.79.0-r0)\n(3/5) Installing expat (2.2.9-r1)\n(4/5) Installing pcre2 (10.34-r1)\n(5/5) Installing git (2.24.4-r0)\nExecuting busybox-1.31.1-r9.trigger\nOK: 22 MiB in 20 packages\nRemoving intermediate container aa4b376f1b0c\n ---\u003e d83e5f976db5\nStep 7/8 : RUN go build -o main\n ---\u003e Running in bbcb8f4d4edf\n\u001b[91mgo: downloading github.com/dadosjusbr/executor v1.0.0\n\u001b[0mRemoving intermediate container bbcb8f4d4edf\n ---\u003e 3b0d24dba7f8\nStep 8/8 : ENTRYPOINT [\"./main\"]\n ---\u003e Running in e717301cfd9a\nRemoving intermediate container e717301cfd9a\n ---\u003e 43c5dc00da2a\nSuccessfully built 43c5dc00da2a\nSuccessfully tagged example-stage-go:latest\n",
    "stderr": "",
    "cmd": "docker build -t example-stage-go .",
    "cmdDir": "//tmp/dadosjusbr-executor398619108/dadosjusbr/example-stage-go",
    "status": 0,
    "env": [
     "COLORTERM=truecolor",
     "LANGUAGE=pt_BR:pt:en",
     "XAUTHORITY=/run/user/1000/gdm/Xauthority",
     "LANG=pt_BR.UTF-8",
     "LS_COLORS=rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.zst=01;31:*.tzst=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.wim=01;31:*.swm=01;31:*.dwm=01;31:*.esd=01;31:*.jpg=01;35:*.jpeg=01;35:*.mjpg=01;35:*.mjpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:",
     "TERM=xterm-256color",
     "DISPLAY=:0",
     "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin",
     "MAIL=/var/mail/root",
     "LOGNAME=root",
     "USER=root",
     "HOME=/root",
     "SHELL=/bin/bash",
     "SUDO_COMMAND=./tutorial",
     "SUDO_USER=daniel",
     "SUDO_UID=1000",
     "SUDO_GID=1000"
    ]
   },
   "runResult": {
    "stdin": "",
    "stdout": "[{\"TPF_Desc\":\"FOLHA 13 ANIVERSARIO          \",\"Nome\":\"ADEZIA LIMA DE CARVALHO                                                         \",\"Cargo\":\"PROMOTOR DE 3ª                          \",\"Lotação\":\"PROMOTORES DE JUSTICA                   \",\"RemuneracaoCargoEfetivo\":0.00,\"OutrasVerbasRemuneratoriasLegaisJudiciais\":0.00,\"FuncaoConfiancaCargoComissao\":0.00,\"GratificacaoNatalina\":33689.16,\"Ferias\":0.00,\"AbonoPermanencia\":0.00,\"TotalRendimentosBruto\":33689.16,\"ContribuicaoPrevidenciaria\":4716.48,\"ImpostoRenda\":7098.13,\"RetencaoTetoConstitucional\":0.00,\"Total_Descontos\":11814.61,\"TOTAL_Liquido\":21874.55,\"Aux_Alimentacao\":0.00,\"Aux_Transporte\":0.00,\"Aux_Moradia\":0.00,\"Aux_FeriasIndenizadas\":0.00,\"Aux_FeriasIndenizadasEstagio\":0.00,\"Insalubridade\":0.00,\"_RemuneracaoLei6773\":0.00,\"RemuneracaoLei6818\":0.00,\"DifEntrancia\":0.00,\"RemuneracaoLei6773/Ato9/2012\":0.00,\"Remuneracao/Ato9/112018\":0.00,\"CoordGruposTrabalho\":0.00,\"ParticComissaoProjetos\":0.00,\"RemunChefiaDirecaoAsses\":0.00}]\n",
    "stderr": "",
    "cmd": "docker run -i -v dadosjusbr:/output --rm --env URL=\"https://raw.githubusercontent.com/dadosjusbr/coletores/master/mpal/src/output_test/membros_ativos-6-2021.json\" --env OUTPUT_FOLDER=\"/output\" example-stage-go",
    "cmdDir": "//tmp/dadosjusbr-executor398619108/dadosjusbr/example-stage-go",
    "status": 0,
    "env": [
     "COLORTERM=truecolor",
     "LANGUAGE=pt_BR:pt:en",
     "XAUTHORITY=/run/user/1000/gdm/Xauthority",
     "LANG=pt_BR.UTF-8",
     "LS_COLORS=rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.zst=01;31:*.tzst=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.wim=01;31:*.swm=01;31:*.dwm=01;31:*.esd=01;31:*.jpg=01;35:*.jpeg=01;35:*.mjpg=01;35:*.mjpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:",
     "TERM=xterm-256color",
     "DISPLAY=:0",
     "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin",
     "MAIL=/var/mail/root",
     "LOGNAME=root",
     "USER=root",
     "HOME=/root",
     "SHELL=/bin/bash",
     "SUDO_COMMAND=./tutorial",
     "SUDO_USER=daniel",
     "SUDO_UID=1000",
     "SUDO_GID=1000"
    ]
   }
  },
  {
   "stage": {
    "name": "Convert the Dadosjusbr json to csv",
    "dir": "/home/daniel/repos/dadosjubsr/executor/tutorial/stage-python",
    "repo": "",
    "base-dir": "",
    "build-env": null,
    "run-env": {
     "OUTPUT_FOLDER": "/output"
    }
   },
   "commit": "",
   "start": "2021-09-20T21:27:58.947032655-03:00",
   "end": "2021-09-20T21:28:00.078091536-03:00",
   "buildResult": {
    "stdin": "",
    "stdout": "Sending build context to Docker daemon  4.096kB\r\r\nStep 1/7 : FROM python:3.7.2-slim\n ---\u003e f46a51a4d255\nStep 2/7 : RUN mkdir /output\n ---\u003e Using cache\n ---\u003e d93270e7590b\nStep 3/7 : WORKDIR /app\n ---\u003e Using cache\n ---\u003e af08aa630c2f\nStep 4/7 : COPY . .\n ---\u003e Using cache\n ---\u003e 78381b5f0afd\nStep 5/7 : RUN pip install --upgrade pip\n ---\u003e Using cache\n ---\u003e b002abfff25e\nStep 6/7 : RUN pip install --no-cache-dir -r requirements.txt\n ---\u003e Using cache\n ---\u003e abf292341af0\nStep 7/7 : CMD [\"python\", \"./script.py\"]\n ---\u003e Using cache\n ---\u003e 9374a16b41a4\nSuccessfully built 9374a16b41a4\nSuccessfully tagged stage-python:latest\n",
    "stderr": "",
    "cmd": "docker build -t stage-python .",
    "cmdDir": "//home/daniel/repos/dadosjubsr/executor/tutorial/stage-python",
    "status": 0,
    "env": [
     "COLORTERM=truecolor",
     "LANGUAGE=pt_BR:pt:en",
     "XAUTHORITY=/run/user/1000/gdm/Xauthority",
     "LANG=pt_BR.UTF-8",
     "LS_COLORS=rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.zst=01;31:*.tzst=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.wim=01;31:*.swm=01;31:*.dwm=01;31:*.esd=01;31:*.jpg=01;35:*.jpeg=01;35:*.mjpg=01;35:*.mjpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:",
     "TERM=xterm-256color",
     "DISPLAY=:0",
     "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin",
     "MAIL=/var/mail/root",
     "LOGNAME=root",
     "USER=root",
     "HOME=/root",
     "SHELL=/bin/bash",
     "SUDO_COMMAND=./tutorial",
     "SUDO_USER=daniel",
     "SUDO_UID=1000",
     "SUDO_GID=1000"
    ]
   },
   "runResult": {
    "stdin": "[{\"TPF_Desc\":\"FOLHA 13 ANIVERSARIO          \",\"Nome\":\"ADEZIA LIMA DE CARVALHO                                                         \",\"Cargo\":\"PROMOTOR DE 3ª                          \",\"Lotação\":\"PROMOTORES DE JUSTICA                   \",\"RemuneracaoCargoEfetivo\":0.00,\"OutrasVerbasRemuneratoriasLegaisJudiciais\":0.00,\"FuncaoConfiancaCargoComissao\":0.00,\"GratificacaoNatalina\":33689.16,\"Ferias\":0.00,\"AbonoPermanencia\":0.00,\"TotalRendimentosBruto\":33689.16,\"ContribuicaoPrevidenciaria\":4716.48,\"ImpostoRenda\":7098.13,\"RetencaoTetoConstitucional\":0.00,\"Total_Descontos\":11814.61,\"TOTAL_Liquido\":21874.55,\"Aux_Alimentacao\":0.00,\"Aux_Transporte\":0.00,\"Aux_Moradia\":0.00,\"Aux_FeriasIndenizadas\":0.00,\"Aux_FeriasIndenizadasEstagio\":0.00,\"Insalubridade\":0.00,\"_RemuneracaoLei6773\":0.00,\"RemuneracaoLei6818\":0.00,\"DifEntrancia\":0.00,\"RemuneracaoLei6773/Ato9/2012\":0.00,\"Remuneracao/Ato9/112018\":0.00,\"CoordGruposTrabalho\":0.00,\"ParticComissaoProjetos\":0.00,\"RemunChefiaDirecaoAsses\":0.00}]\n",
    "stdout": "",
    "stderr": "",
    "cmd": "docker run -i -v dadosjusbr:/output --rm --env OUTPUT_FOLDER=\"/output\" stage-python",
    "cmdDir": "//home/daniel/repos/dadosjubsr/executor/tutorial/stage-python",
    "status": 0,
    "env": [
     "COLORTERM=truecolor",
     "LANGUAGE=pt_BR:pt:en",
     "XAUTHORITY=/run/user/1000/gdm/Xauthority",
     "LANG=pt_BR.UTF-8",
     "LS_COLORS=rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.zst=01;31:*.tzst=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.wim=01;31:*.swm=01;31:*.dwm=01;31:*.esd=01;31:*.jpg=01;35:*.jpeg=01;35:*.mjpg=01;35:*.mjpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:",
     "TERM=xterm-256color",
     "DISPLAY=:0",
     "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin",
     "MAIL=/var/mail/root",
     "LOGNAME=root",
     "USER=root",
     "HOME=/root",
     "SHELL=/bin/bash",
     "SUDO_COMMAND=./tutorial",
     "SUDO_USER=daniel",
     "SUDO_UID=1000",
     "SUDO_GID=1000"
    ]
   }
  }
 ],
 "start": "2021-09-20T21:27:47.357009506-03:00",
 "final": "2021-09-20T21:28:00.106597287-03:00",
 "status": "OK"
}
```
