// Copyright 2021 - 2022 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tree

// the CREATE TRIGGER statement.
type Create_trigger struct {
	statementImpl
	trigger_name  TableExpr
	trigger_time  IdentifierList
	trigger_event IdentifierList
	Rows          *Select
}

func (node *Create_trigger) Format(ctx *FmtCtx) {
	ctx.WriteString("create trigger")
	node.trigger_name.Format(ctx)

	if node.trigger_time != nil {
		ctx.WriteString(" partition(")
		node.trigger_event.Format(ctx)
		ctx.WriteByte(')')
	}

	if node.trigger_event != nil {
		ctx.WriteString(" (")
		node.trigger_event.Format(ctx)
		ctx.WriteByte(')')
	}
	if node.Rows != nil {
		ctx.WriteByte(' ')
		node.Rows.Format(ctx)
	}
}

func NewReplace(t TableExpr, c IdentifierList, r *Select, p IdentifierList) *Create_trigger {
	return &Create_trigger{
		trigger_name:  t,
		trigger_time:  c,
		Rows:          r,
		trigger_event: p,
	}
}
