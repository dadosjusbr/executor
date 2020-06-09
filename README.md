# executor

Biblioteca go que permite a execução de um pipeline DadosjusBR.

## O que é um Pipeline DadosjusBR?

É uma sequência de etapas que visam a libertação e padronização de dados do sistema de justiça brasileiro.

As etapas são: 
- Coleta: Extrai informações, consolida e faz a tradução necessária para um formato único do DadosJusBr e baixa todo artefato que for necessário. Mais informações [nesse link](https://github.com/dadosjusbr/coletores).
- Validação: Responsável por fazer validações nos dados de acordo a cada contexto;
- Empacotamento: Responsável por padronizar os dados no formato de datapackages;
- Armazenamento: Responsável por armazenar os dados extraídos, além de versionar também os artefatos baixados e gerados durante a coleta;
