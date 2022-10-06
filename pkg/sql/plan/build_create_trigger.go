package plan

import (
	"github.com/matrixorigin/matrixone/pkg/common/moerr"
	"github.com/matrixorigin/matrixone/pkg/sql/parsers/tree"
)

func build_create_trigger(stmt *tree.Create_trigger, ctx CompilerContext) (p *Plan, err error) {
	// no support replace now
	return nil, moerr.NewNotSupported("Not support create trigger statement")
}
