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

package single

import (
	"bytes"

	"github.com/matrixorigin/matrixone/pkg/common/hashmap"
	"github.com/matrixorigin/matrixone/pkg/common/moerr"
	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/sql/colexec"
	"github.com/matrixorigin/matrixone/pkg/sql/plan"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
)

func String(_ any, buf *bytes.Buffer) {
	buf.WriteString(" single join ")
}

func Prepare(proc *process.Process, arg any) error {
	ap := arg.(*Argument)
	ap.ctr = new(container)
	ap.ctr.inBuckets = make([]uint8, hashmap.UnitLimit)
	ap.ctr.evecs = make([]evalVector, len(ap.Conditions[0]))
	ap.ctr.vecs = make([]*vector.Vector, len(ap.Conditions[0]))
	ap.ctr.bat = batch.NewWithSize(len(ap.Typs))
	ap.ctr.bat.Zs = proc.GetMheap().GetSels()
	for i, typ := range ap.Typs {
		ap.ctr.bat.Vecs[i] = vector.New(typ)
	}
	return nil
}

func Call(idx int, proc *process.Process, arg any) (bool, error) {
	anal := proc.GetAnalyze(idx)
	anal.Start()
	defer anal.Stop()
	ap := arg.(*Argument)
	ctr := ap.ctr
	for {
		switch ctr.state {
		case Build:
			if err := ctr.build(ap, proc, anal); err != nil {
				ctr.state = End
				if ctr.mp != nil {
					ctr.mp.Free()
				}
				return true, err
			}
			ctr.state = Probe
		case Probe:
			bat := <-proc.Reg.MergeReceivers[0].Ch
			if bat == nil {
				ctr.state = End
				if ctr.mp != nil {
					ctr.mp.Free()
				}
				if ctr.bat != nil {
					ctr.bat.Clean(proc.GetMheap())
				}
				continue
			}
			if bat.Length() == 0 {
				continue
			}
			if ctr.bat.Length() == 0 {
				if err := ctr.emptyProbe(bat, ap, proc, anal); err != nil {
					ctr.state = End
					if ctr.mp != nil {
						ctr.mp.Free()
					}
					proc.SetInputBatch(nil)
					return true, err
				}

			} else {
				if err := ctr.probe(bat, ap, proc, anal); err != nil {
					ctr.state = End
					if ctr.mp != nil {
						ctr.mp.Free()
					}
					proc.SetInputBatch(nil)
					return true, err
				}
			}
			return false, nil
		default:
			proc.SetInputBatch(nil)
			return true, nil
		}
	}
}

func (ctr *container) build(ap *Argument, proc *process.Process, anal process.Analyze) error {
	bat := <-proc.Reg.MergeReceivers[1].Ch
	if bat != nil {
		ctr.bat = bat
		ctr.mp = bat.Ht.(*hashmap.JoinMap).Dup()
	}
	return nil
}

func (ctr *container) emptyProbe(bat *batch.Batch, ap *Argument, proc *process.Process, anal process.Analyze) error {
	defer bat.Clean(proc.GetMheap())
	anal.Input(bat)
	rbat := batch.NewWithSize(len(ap.Result))
	count := bat.Length()
	for i, rp := range ap.Result {
		if rp.Rel == 0 {
			rbat.Vecs[i] = bat.Vecs[rp.Pos]
			bat.Vecs[rp.Pos] = nil
		} else {
			rbat.Vecs[i] = vector.NewConstNull(ctr.bat.Vecs[rp.Pos].Typ, count)
		}
	}
	rbat.Zs = bat.Zs
	bat.Zs = nil
	anal.Output(rbat)
	proc.SetInputBatch(rbat)
	return nil
}

func (ctr *container) probe(bat *batch.Batch, ap *Argument, proc *process.Process, anal process.Analyze) error {
	defer bat.Clean(proc.GetMheap())
	anal.Input(bat)
	rbat := batch.NewWithSize(len(ap.Result))
	for i, rp := range ap.Result {
		if rp.Rel != 0 {
			rbat.Vecs[i] = vector.New(ctr.bat.Vecs[rp.Pos].Typ)
		}
	}
	if err := ctr.evalJoinCondition(bat, ap.Conditions[0], proc); err != nil {
		return err
	}
	defer ctr.freeJoinCondition(proc)
	count := bat.Length()
	mSels := ctr.mp.Sels()
	itr := ctr.mp.Map().NewIterator()
	for i := 0; i < count; i += hashmap.UnitLimit {
		n := count - i
		if n > hashmap.UnitLimit {
			n = hashmap.UnitLimit
		}
		copy(ctr.inBuckets, hashmap.OneUInt8s)
		vals, zvals := itr.Find(i, n, ctr.vecs, ctr.inBuckets)
		for k := 0; k < n; k++ {
			if ctr.inBuckets[k] == 0 {
				continue
			}
			if zvals[k] == 0 || vals[k] == 0 {
				for j, rp := range ap.Result {
					if rp.Rel != 0 {
						if err := vector.UnionNull(rbat.Vecs[j], ctr.bat.Vecs[rp.Pos], proc.GetMheap()); err != nil {
							rbat.Clean(proc.GetMheap())
							return err
						}
					}
				}
				continue
			}
			idx := 0
			matched := false
			sels := mSels[vals[k]-1]
			if ap.Cond != nil {
				for j, sel := range sels {
					vec, err := colexec.JoinFilterEvalExprInBucket(bat, ctr.bat, i+k, int(sel), proc, ap.Cond)
					if err != nil {
						return err
					}
					bs := vec.Col.([]bool)
					if bs[0] {
						if matched {
							return moerr.NewInternalError("scalar subquery returns more than 1 row")
						}
						matched = true
						idx = j
					}
					vec.Free(proc.Mp())
				}
			} else if len(sels) > 1 {
				return moerr.NewInternalError("scalar subquery returns more than 1 row")
			}
			if ap.Cond != nil && !matched {
				for j, rp := range ap.Result {
					if rp.Rel != 0 {
						if err := vector.UnionNull(rbat.Vecs[j], ctr.bat.Vecs[rp.Pos], proc.GetMheap()); err != nil {
							rbat.Clean(proc.GetMheap())
							return err
						}
					}
				}
				continue
			}
			sel := sels[idx]
			for j, rp := range ap.Result {
				if rp.Rel != 0 {
					if err := vector.UnionOne(rbat.Vecs[j], ctr.bat.Vecs[rp.Pos], sel, proc.GetMheap()); err != nil {
						rbat.Clean(proc.GetMheap())
						return err
					}
				}
			}
		}
	}
	for i, rp := range ap.Result {
		if rp.Rel == 0 {
			rbat.Vecs[i] = bat.Vecs[rp.Pos]
			bat.Vecs[rp.Pos] = nil
		}
	}
	rbat.Zs = bat.Zs
	bat.Zs = nil
	rbat.ExpandNulls()
	anal.Output(rbat)
	proc.SetInputBatch(rbat)
	return nil
}

func (ctr *container) evalJoinCondition(bat *batch.Batch, conds []*plan.Expr, proc *process.Process) error {
	for i, cond := range conds {
		vec, err := colexec.EvalExpr(bat, proc, cond)
		if err != nil || vec.ConstExpand(proc.GetMheap()) == nil {
			for j := 0; j < i; j++ {
				if ctr.evecs[j].needFree {
					vector.Clean(ctr.evecs[j].vec, proc.GetMheap())
				}
			}
			return err
		}
		ctr.vecs[i] = vec
		ctr.evecs[i].vec = vec
		ctr.evecs[i].needFree = true
		for j := range bat.Vecs {
			if bat.Vecs[j] == vec {
				ctr.evecs[i].needFree = false
				break
			}
		}
	}
	return nil
}

func (ctr *container) freeJoinCondition(proc *process.Process) {
	for i := range ctr.evecs {
		if ctr.evecs[i].needFree {
			ctr.evecs[i].vec.Free(proc.GetMheap())
		}
	}
}
