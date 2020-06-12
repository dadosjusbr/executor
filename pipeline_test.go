package executor

import (
	"testing"

	"github.com/matryer/is"
)

func TestSetup(t *testing.T) {
	is := is.New(t)
	dir := "/home/lsp/projetos/go/src/github.com/dadosjusbr/coletores/mppb"
	is.NoErr(setup(dir))
}
