// Copyright 2021 Matrix Origin
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

package vm

const (
	Top = iota
	Join
	Semi
	Left
	Limit
	Merge
	Order
	Group
	Output
	Offset
	Product
	Restrict
	Dispatch
	Connector
	Projection
	Anti
	Single
	Mark

	LoopJoin
	LoopLeft
	LoopSemi
	LoopAnti
	LoopSingle

	MergeTop
	MergeLimit
	MergeOrder
	MergeGroup
	MergeOffset

	Deletion
	Insert
	Update
	External

	Minus
	Intersect
	IntersectAll

	HashBuild

	Unnest
)

// Instruction contains relational algebra
type Instruction struct {
	// Op specified the operator code of an instruction.
	Op int
	// Idx specified the anaylze information index.
	Idx int
	// Arg contains the operand of this instruction.
	Arg any
}

type Instructions []Instruction
