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

package vector

import (
	"bytes"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/matrixorigin/matrixone/pkg/common/moerr"
	"github.com/matrixorigin/matrixone/pkg/container/nulls"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/vectorize/shuffle"
	"github.com/matrixorigin/matrixone/pkg/vm/mheap"
)

// XXX Moved vector from types.go to vector.go
// XXX Deleted vector interface, which was commented out and outdated anyway.
/* Vector vector
 * origin true:
 * 				count || type || bitmap size || bitmap || vector
 * origin false:
 *  			count || vector
 */
type Vector struct {
	// XXX There was Ref and Link, from the impl, it is totally wrong stuff.
	// Removed.
	Typ types.Type
	Col interface{}  // column data, encoded Data
	Nsp *nulls.Nulls // nulls list

	original bool
	// data of fixed length element, in case of varlen, the Varlena
	data []byte
	// area for holding large strings.
	area []byte

	// some attributes for const vector (a vector with a lot of rows of a same const value)
	isConst bool
	length  int
}

func (v *Vector) Length() int {
	return Length(v)
}

func (v *Vector) ScalarLength() int {
	if !v.isConst {
		panic("Getting scalar length of non const vector.")
	}
	return v.length
}

func (v *Vector) SetScalarLength(length int) {
	if !v.isConst {
		panic("Setting length to non const vector.")
	}
	v.length = length
}

func (v *Vector) IsOriginal() bool {
	return v.original
}

func (v *Vector) SetOriginal(status bool) {
	v.original = status
}

func DecodeFixedCol[T types.FixedSizeT](v *Vector, sz int) []T {
	return types.DecodeFixedSlice[T](v.data, sz)
}

// GetFixedVector decode data and return decoded []T.
// For const/scalar vector we expand and return newly allocated slice.
func GetFixedVectorValues[T types.FixedSizeT](v *Vector) []T {
	if v.isConst {
		cols := MustTCols[T](v)
		vs := make([]T, v.Length())
		for i := range vs {
			vs[i] = cols[0]
		}
		return vs
	}
	return MustTCols[T](v)
}

func GetStrVectorValues(v *Vector) []string {
	if v.isConst {
		cols := MustTCols[types.Varlena](v)
		ss := cols[0].GetString(v.area)
		vs := make([]string, v.Length())
		for i := range vs {
			vs[i] = ss
		}
		return vs
	}
	return MustStrCols(v)
}

func GetBytesVectorValues(v *Vector) [][]byte {
	if v.isConst {
		cols := MustTCols[types.Varlena](v)
		ss := cols[0].GetByteSlice(v.area)
		vs := make([][]byte, v.Length())
		for i := range vs {
			vs[i] = ss
		}
		return vs
	}
	return MustBytesCols(v)
}

// XXX A huge hammer, get rid of any typing and totally depends on v.Col
// We should really not using this one but it is wide spread already.
func GetColumn[T any](v *Vector) []T {
	return v.Col.([]T)
}

// XXX Compatibility: how many aliases do we need ...
func GetStrColumn(v *Vector) []string {
	return GetStrVectorValues(v)
}

// Get Value at index
func GetValueAt[T types.FixedSizeT](v *Vector, idx int64) T {
	return MustTCols[T](v)[idx]
}

func GetValueAtOrZero[T types.FixedSizeT](v *Vector, idx int64) T {
	var zt T
	ts := MustTCols[T](v)
	if int64(len(ts)) <= idx {
		return zt
	}
	return ts[idx]
}

// Get the pointer to idx-th fixed size entry.
func GetPtrAt(v *Vector, idx int64) unsafe.Pointer {
	return unsafe.Pointer(&v.data[idx*int64(v.GetType().TypeSize())])
}

// Raw version, get from v.data.   Adopt python convention and
// neg idx means counting from end, that is, -1 means last element.
func (v *Vector) getRawValueAt(idx int64) []byte {
	tlen := int64(v.GetType().TypeSize())
	dlen := int64(len(v.data))
	if idx >= 0 {
		if dlen < (idx+1)*tlen {
			panic("vector invalid index access")
		}
		return v.data[idx*tlen : idx*tlen+tlen]
	} else {
		start := dlen + tlen*idx
		end := start + tlen
		if start < 0 {
			panic("vector invalid index access")
		}
		return v.data[start:end]
	}
}

func (v *Vector) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	if v.isConst {
		i64 := int64(v.ScalarLength())
		buf.WriteByte(1)
		buf.Write(types.EncodeInt64(&i64))
	} else {
		buf.WriteByte(0)
		// length, even not used, let's fill it.
		i64 := int64(0)
		buf.Write(types.EncodeInt64(&i64))
	}
	data, err := v.Show()
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	return buf.Bytes(), nil
}

func (v *Vector) UnmarshalBinary(data []byte) error {
	if data[0] == 1 {
		v.isConst = true
		data = data[1:]
		v.SetScalarLength(int(types.DecodeInt64(data[:8])))
		data = data[8:]
	} else {
		data = data[1:]
		// skip 0
		data = data[8:]
	}
	return v.Read(data)
}

// Size of data, I think this function is inherently broken.  This
// Size is not meaningful other than used in (approximate) memory accounting.
func (v *Vector) Size() int {
	return len(v.data) + len(v.area)
}

func (v *Vector) GetArea() []byte {
	return v.area
}

func (v *Vector) GetType() types.Type {
	return v.Typ
}

func (v *Vector) GetNulls() *nulls.Nulls {
	return v.Nsp
}

func (v *Vector) GetBytes(i int64) []byte {
	bs := MustTCols[types.Varlena](v)
	return bs[i].GetByteSlice(v.area)
}

func (v *Vector) GetString(i int64) string {

	bs := MustTCols[types.Varlena](v)
	return bs[i].GetString(v.area)
}

func (v *Vector) FillDefaultValue() {
	if !nulls.Any(v.Nsp) || len(v.data) == 0 {
		return
	}
	switch v.Typ.Oid {
	case types.T_bool:
		fillDefaultValue[bool](v)
	case types.T_int8:
		fillDefaultValue[int8](v)
	case types.T_int16:
		fillDefaultValue[int16](v)
	case types.T_int32:
		fillDefaultValue[int32](v)
	case types.T_int64:
		fillDefaultValue[int64](v)
	case types.T_uint8:
		fillDefaultValue[uint8](v)
	case types.T_uint16:
		fillDefaultValue[uint16](v)
	case types.T_uint32:
		fillDefaultValue[uint32](v)
	case types.T_uint64:
		fillDefaultValue[uint64](v)
	case types.T_float32:
		fillDefaultValue[float32](v)
	case types.T_float64:
		fillDefaultValue[float64](v)
	case types.T_date:
		fillDefaultValue[types.Date](v)
	case types.T_datetime:
		fillDefaultValue[types.Datetime](v)
	case types.T_timestamp:
		fillDefaultValue[types.Timestamp](v)
	case types.T_decimal64:
		fillDefaultValue[types.Decimal64](v)
	case types.T_decimal128:
		fillDefaultValue[types.Decimal128](v)
	case types.T_uuid:
		fillDefaultValue[types.Uuid](v)
	case types.T_TS:
		fillDefaultValue[types.TS](v)
	case types.T_Rowid:
		fillDefaultValue[types.Rowid](v)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		fillDefaultValue[types.Varlena](v)
	default:
		panic("unsupported type in FillDefaultValue")
	}
}

func (v *Vector) ToConst(row int) *Vector {
	if v.isConst {
		return v
	}
	switch v.Typ.Oid {
	case types.T_bool:
		return toConstVector[bool](v, row)
	case types.T_int8:
		return toConstVector[int8](v, row)
	case types.T_int16:
		return toConstVector[int16](v, row)
	case types.T_int32:
		return toConstVector[int32](v, row)
	case types.T_int64:
		return toConstVector[int64](v, row)
	case types.T_uint8:
		return toConstVector[uint8](v, row)
	case types.T_uint16:
		return toConstVector[uint16](v, row)
	case types.T_uint32:
		return toConstVector[uint32](v, row)
	case types.T_uint64:
		return toConstVector[uint64](v, row)
	case types.T_float32:
		return toConstVector[float32](v, row)
	case types.T_float64:
		return toConstVector[float64](v, row)
	case types.T_date:
		return toConstVector[types.Date](v, row)
	case types.T_datetime:
		return toConstVector[types.Datetime](v, row)
	case types.T_timestamp:
		return toConstVector[types.Timestamp](v, row)
	case types.T_decimal64:
		return toConstVector[types.Decimal64](v, row)
	case types.T_decimal128:
		return toConstVector[types.Decimal128](v, row)
	case types.T_uuid:
		return toConstVector[types.Uuid](v, row)
	case types.T_TS:
		return toConstVector[types.TS](v, row)
	case types.T_Rowid:
		return toConstVector[types.Rowid](v, row)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		if nulls.Contains(v.Nsp, uint64(row)) {
			return NewConstNull(v.GetType(), 1)
		}
		bs := v.GetBytes(int64(row))
		return NewConstBytes(v.Typ, 1, bs)
	}
	return nil
}

func (v *Vector) ConstExpand(m *mheap.Mheap) *Vector {
	if !v.isConst {
		return v
	}
	if v.IsScalarNull() {
		vlen := uint64(v.ScalarLength())
		nulls.AddRange(v.Nsp, 0, vlen)
		return v
	}

	switch v.Typ.Oid {
	case types.T_bool:
		expandVector[bool](v, 1, m)
	case types.T_int8:
		expandVector[int8](v, 1, m)
	case types.T_int16:
		expandVector[int16](v, 2, m)
	case types.T_int32:
		expandVector[int32](v, 4, m)
	case types.T_int64:
		expandVector[int64](v, 8, m)
	case types.T_uint8:
		expandVector[uint8](v, 1, m)
	case types.T_uint16:
		expandVector[uint16](v, 2, m)
	case types.T_uint32:
		expandVector[uint32](v, 4, m)
	case types.T_uint64:
		expandVector[uint64](v, 8, m)
	case types.T_float32:
		expandVector[float32](v, 4, m)
	case types.T_float64:
		expandVector[float64](v, 8, m)
	case types.T_date:
		expandVector[types.Date](v, 4, m)
	case types.T_datetime:
		expandVector[types.Datetime](v, 8, m)
	case types.T_timestamp:
		expandVector[types.Timestamp](v, 8, m)
	case types.T_decimal64:
		expandVector[types.Decimal64](v, 8, m)
	case types.T_decimal128:
		expandVector[types.Decimal128](v, 16, m)
	case types.T_uuid:
		expandVector[types.Uuid](v, 16, m)
	case types.T_TS:
		expandVector[types.TS](v, types.TxnTsSize, m)
	case types.T_Rowid:
		expandVector[types.Rowid](v, types.RowidSize, m)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		expandVector[types.Varlena](v, types.VarlenaSize, m)
	}
	v.isConst = false
	return v
}

func (v *Vector) TryExpandNulls(n int) {
	nulls.TryExpand(v.Nsp, n)
}

func fillDefaultValue[T types.FixedSizeT](v *Vector) {
	var dv T
	col := v.Col.([]T)
	rows := v.Nsp.Np.ToArray()
	for _, row := range rows {
		col[row] = dv
	}
	v.Col = col
}

func toConstVector[T types.FixedSizeT](v *Vector, row int) *Vector {
	if nulls.Contains(v.Nsp, uint64(row)) {
		return NewConstNull(v.Typ, 1)
	} else {
		val := GetValueAt[T](v, int64(row))
		return NewConstFixed(v.Typ, 1, val)
	}
}

// expandVector is used only in expand const vector.
func expandVector[T any](v *Vector, sz int, m *mheap.Mheap) *Vector {
	data, err := mheap.Alloc(m, int64(v.ScalarLength()*sz))
	if err != nil {
		return nil
	}
	vs := types.DecodeFixedSlice[T](data, sz)
	if nulls.Any(v.Nsp) {
		for i := 0; i < v.ScalarLength(); i++ {
			nulls.Add(v.Nsp, uint64(i))
		}
	} else {
		val := v.Col.([]T)[0]
		for i := 0; i < v.ScalarLength(); i++ {
			vs[i] = val
		}
	}
	v.Col = vs
	v.data = data[:len(vs)*sz]
	return v
}

func NewWithData(typ types.Type, data []byte, col interface{}, nsp *nulls.Nulls) *Vector {
	v := &Vector{
		Nsp:  nsp,
		Typ:  typ,
		Col:  col,
		data: data,
	}
	if col == nil {
		v.colFromData()
	}
	return v
}

func NewWithStrings(typ types.Type, vals []string, nsp *nulls.Nulls, m *mheap.Mheap) *Vector {
	vec := New(typ)
	nulls.Set(vec.Nsp, nsp)
	AppendString(vec, vals, m)
	return vec
}

func NewWithBytes(typ types.Type, vals [][]byte, nsp *nulls.Nulls, m *mheap.Mheap) *Vector {
	vec := New(typ)
	nulls.Set(vec.Nsp, nsp)
	AppendBytes(vec, vals, m)
	return vec
}

func NewWithFixed[T types.FixedSizeT](typ types.Type, vals []T, nsp *nulls.Nulls, m *mheap.Mheap) *Vector {
	vec := New(typ)
	nulls.Set(vec.Nsp, nsp)
	AppendFixed(vec, vals, m)
	return vec
}

func New(typ types.Type) *Vector {
	return NewWithData(typ, []byte{}, nil, &nulls.Nulls{})
}

func NewOriginal(typ types.Type) *Vector {
	vec := NewWithData(typ, []byte{}, nil, &nulls.Nulls{})
	vec.original = true
	return vec
}

func NewWithNspSize(typ types.Type, n int64) *Vector {
	return NewWithData(typ, []byte{}, nil, nulls.NewWithSize(int(n)))
}

func NewConst(typ types.Type, length int) *Vector {
	v := New(typ)
	v.isConst = true
	v.initConst(typ)
	v.length = length
	return v
}

func NewConstNull(typ types.Type, length int) *Vector {
	v := New(typ)
	v.isConst = true
	v.initConst(typ)
	nulls.Add(v.Nsp, 0)
	v.length = length
	return v
}

func NewConstFixed[T types.FixedSizeT](typ types.Type, length int, val T) *Vector {
	v := NewConst(typ, length)
	// TODO: memory accounting
	v.Append(val, false, nil)
	return v
}

func NewConstString(typ types.Type, length int, val string) *Vector {
	v := NewConst(typ, length)
	// XXX memory accounting?
	SetStringAt(v, 0, val, nil)
	return v
}

func NewConstBytes(typ types.Type, length int, val []byte) *Vector {
	v := NewConst(typ, length)
	// XXX memory accounting?
	SetBytesAt(v, 0, val, nil)
	return v
}

func (v *Vector) initConst(typ types.Type) {
	switch typ.Oid {
	case types.T_bool:
		v.Col = []bool{false}
	case types.T_int8:
		v.Col = []int8{0}
	case types.T_int16:
		v.Col = []int16{0}
	case types.T_int32:
		v.Col = []int32{0}
	case types.T_int64:
		v.Col = []int64{0}
	case types.T_uint8:
		v.Col = []uint8{0}
	case types.T_uint16:
		v.Col = []uint16{0}
	case types.T_uint32:
		v.Col = []uint32{0}
	case types.T_uint64:
		v.Col = []uint64{0}
	case types.T_float32:
		v.Col = []float32{0}
	case types.T_float64:
		v.Col = []float64{0}
	case types.T_date:
		v.Col = make([]types.Date, 1)
	case types.T_datetime:
		v.Col = make([]types.Datetime, 1)
	case types.T_timestamp:
		v.Col = make([]types.Timestamp, 1)
	case types.T_decimal64:
		v.Col = make([]types.Decimal64, 1)
	case types.T_decimal128:
		v.Col = make([]types.Decimal128, 1)
	case types.T_uuid:
		v.Col = make([]types.Uuid, 1)
	case types.T_TS:
		v.Col = make([]types.TS, 1)
	case types.T_Rowid:
		v.Col = make([]types.Rowid, 1)
	case types.T_char, types.T_varchar, types.T_blob, types.T_json:
		v.Col = make([]types.Varlena, 1)
	}
}

// IsScalar return true if the vector means a scalar value.
// e.g.
//
//	a + 1, and 1's vector will return true
func (v *Vector) IsScalar() bool {
	return v.isConst
}
func (v *Vector) IsConst() bool {
	return v.isConst
}

// MakeScalar converts a vector to a scalar vec of length.
func (v *Vector) MakeScalar(length int) {
	if v.isConst {
		v.length = length
	} else {
		if v.Length() != 1 {
			panic("make scalar called on a vec")
		}
		v.isConst = true
		v.length = length
	}
}

// IsScalarNull return true if the vector means a scalar Null.
// e.g.
//
//	a + Null, and the vector of right part will return true
func (v *Vector) IsScalarNull() bool {
	return v.isConst && v.Nsp != nil && nulls.Contains(v.Nsp, 0)
}

// XXX aliases ...
func (v *Vector) ConstVectorIsNull() bool {
	return v.IsScalarNull()
}

func (v *Vector) Free(m *mheap.Mheap) {
	if v.original {
		// XXX: Should we panic, or this is really an Noop?
		return
	}
	if !v.IsConst() {
		// const vector's data & area allocate with nil mheap
		// so we can't free it by using mheap.
		mheap.Free(m, v.data)
		mheap.Free(m, v.area)
	}
	v.data = []byte{}
	v.colFromData()
	v.area = nil
}

func (v *Vector) FreeOriginal(m *mheap.Mheap) {
	if v.original {
		mheap.Free(m, v.data)
		v.data = []byte{}
		v.colFromData()
		mheap.Free(m, v.area)
		v.area = nil
		return
	}
	panic("force original tries to free non-orignal vec")
}

func appendOne[T types.FixedSizeT](v *Vector, w T, isNull bool, m *mheap.Mheap) error {
	if err := v.extend(1, m); err != nil {
		return err
	}
	col := MustTCols[T](v)
	pos := len(col) - 1
	if isNull {
		nulls.Add(v.Nsp, uint64(pos))
	} else {
		col[pos] = w
	}
	return nil
}

func appendOneBytes(v *Vector, bs []byte, isNull bool, m *mheap.Mheap) error {
	var err error
	var va types.Varlena
	if isNull {
		return appendOne(v, va, true, m)
	} else {
		va, v.area, err = types.BuildVarlena(bs, v.area, m)
		if err != nil {
			return err
		}
		return appendOne(v, va, false, m)
	}
}

func (v *Vector) Append(w any, isNull bool, m *mheap.Mheap) error {
	switch v.Typ.Oid {
	case types.T_bool:
		return appendOne(v, w.(bool), isNull, m)
	case types.T_int8:
		return appendOne(v, w.(int8), isNull, m)
	case types.T_int16:
		return appendOne(v, w.(int16), isNull, m)
	case types.T_int32:
		return appendOne(v, w.(int32), isNull, m)
	case types.T_int64:
		return appendOne(v, w.(int64), isNull, m)
	case types.T_uint8:
		return appendOne(v, w.(uint8), isNull, m)
	case types.T_uint16:
		return appendOne(v, w.(uint16), isNull, m)
	case types.T_uint32:
		return appendOne(v, w.(uint32), isNull, m)
	case types.T_uint64:
		return appendOne(v, w.(uint64), isNull, m)
	case types.T_float32:
		return appendOne(v, w.(float32), isNull, m)
	case types.T_float64:
		return appendOne(v, w.(float64), isNull, m)
	case types.T_date:
		return appendOne(v, w.(types.Date), isNull, m)
	case types.T_datetime:
		return appendOne(v, w.(types.Datetime), isNull, m)
	case types.T_timestamp:
		return appendOne(v, w.(types.Timestamp), isNull, m)
	case types.T_decimal64:
		return appendOne(v, w.(types.Decimal64), isNull, m)
	case types.T_decimal128:
		return appendOne(v, w.(types.Decimal128), isNull, m)
	case types.T_uuid:
		return appendOne(v, w.(types.Uuid), isNull, m)
	case types.T_TS:
		return appendOne(v, w.(types.TS), isNull, m)
	case types.T_Rowid:
		return appendOne(v, w.(types.Rowid), isNull, m)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		if isNull {
			return appendOneBytes(v, nil, true, m)
		}
		wv := w.([]byte)
		return appendOneBytes(v, wv, false, m)
	}
	return nil
}

func Clean(v *Vector, m *mheap.Mheap) {
	v.Free(m)
}

func SetCol(v *Vector, col interface{}) {
	v.Col = col
}

func SetTAt[T types.FixedSizeT](v *Vector, idx int, t T) error {
	// Let it panic if v is not a varlena vec
	vacol := MustTCols[T](v)

	if idx < 0 {
		idx = len(vacol) + idx
	}
	if idx < 0 || idx >= len(vacol) {
		return moerr.NewInternalError("vector idx out of range")
	}
	vacol[idx] = t
	return nil
}

func SetBytesAt(v *Vector, idx int, bs []byte, m *mheap.Mheap) error {
	var va types.Varlena
	var err error
	va, v.area, err = types.BuildVarlena(bs, v.area, m)
	if err != nil {
		return err
	}
	return SetTAt(v, idx, va)
}

func SetStringAt(v *Vector, idx int, bs string, m *mheap.Mheap) error {
	return SetBytesAt(v, idx, []byte(bs), m)
}

// XXX: PreAlloc create a empty v, with enough fixed slots to cap entry.
func PreAlloc(v *Vector, rows, cap int, m *mheap.Mheap) {
	var data []byte
	var err error
	sz := int64(cap * v.GetType().TypeSize())
	if m == nil {
		data = make([]byte, sz)
	} else {
		data, err = mheap.Alloc(m, int64(rows*v.GetType().TypeSize()))
	}

	// XXX: Was just returned and runs defer, which was Huh?   Let me panic.
	if err != nil {
		panic(err)
	}
	v.data = data
	v.setupColFromData(0, rows)
}

func PreAllocType(t types.Type, rows, cap int, m *mheap.Mheap) *Vector {
	vec := New(t)
	PreAlloc(vec, rows, cap, m)
	return vec
}

// XXX Confusing as hell.
func Length(v *Vector) int {
	if !v.isConst {
		return reflect.ValueOf(v.Col).Len()
	}
	return v.ScalarLength()
}

func SetLength(v *Vector, n int) {
	if v.IsScalar() {
		// XXX old code test this one.  Why? || v.Typ.Oid == types.T_any {
		v.SetScalarLength(n)
		return
	}
	SetVectorLength(v, n)
}

func SetVectorLength(v *Vector, n int) {
	end := len(v.data) / v.GetType().TypeSize()
	if n > end {
		panic("extend instead of shink vector")
	}
	nulls.RemoveRange(v.Nsp, uint64(n), uint64(end))
	v.setupColFromData(0, n)
}

// XXX Original code is really confused by what is dup ...
func Dup(v *Vector, m *mheap.Mheap) (*Vector, error) {
	to := Vector{
		Typ: v.Typ,
		Nsp: v.Nsp, // XXX: dude, you do not dup this?
	}

	var err error

	// Copy v.data, note that this should work for Varlena type
	// as because we will copy area next and offset len will stay
	// valid for long varlena.
	if len(v.data) > 0 {
		if to.data, err = mheap.Alloc(m, int64(len(v.data))); err != nil {
			return nil, err
		}
		copy(to.data, v.data)
	}
	if len(v.area) > 0 {
		if to.area, err = mheap.Alloc(m, int64(len(v.area))); err != nil {
			return nil, err
		}
		copy(to.area, v.area)
	}

	nele := len(v.data) / v.GetType().TypeSize()
	to.setupColFromData(0, nele)
	return &to, nil
}

// Window just returns a window out of input and no deep copy.
func Window(v *Vector, start, end int, w *Vector) *Vector {
	w.Typ = v.Typ
	w.Nsp = nulls.Range(v.Nsp, uint64(start), uint64(end), w.Nsp)
	w.data = v.data
	w.area = v.area
	w.setupColFromData(start, end)
	return w
}

func AppendFixed[T types.FixedSizeT](v *Vector, arg []T, m *mheap.Mheap) error {
	var err error
	col := MustTCols[T](v)
	ncol := len(col)
	narg := len(arg)
	if narg == 0 {
		return nil
	}

	nsz := (ncol + narg) * v.GetType().TypeSize()
	v.data, err = m.Grow(v.data, int64(nsz))
	if err != nil {
		return err
	}
	v.colFromData()
	col = MustTCols[T](v)
	copy(col[ncol:ncol+narg], arg)
	return nil
}

func AppendFixedRaw(v *Vector, data []byte) error {
	v.data = append(v.data, data...)
	v.colFromData()
	return nil
}

func AppendBytes(v *Vector, arg [][]byte, m *mheap.Mheap) error {
	var err error
	vas := make([]types.Varlena, len(arg))
	for idx, bs := range arg {
		// XXX we do not track memory usage anymore?
		vas[idx], v.area, err = types.BuildVarlena(bs, v.area, m)
		if err != nil {
			return err
		}
	}
	return AppendFixed(v, vas, m)
}

func AppendString(v *Vector, arg []string, m *mheap.Mheap) error {
	var err error
	vas := make([]types.Varlena, len(arg))
	for idx, bs := range arg {
		// XXX we do not track memory usage anymore?
		vas[idx], v.area, err = types.BuildVarlena([]byte(bs), v.area, m)
		if err != nil {
			return err
		}
	}
	return AppendFixed(v, vas, m)
}

func AppendTuple(v *Vector, arg [][]interface{}) error {
	if v.GetType().IsTuple() {
		return moerr.NewInternalError("append tuple to non tuple vector")
	}
	v.Col = append(v.Col.([][]interface{}), arg...)
	return nil
}

func ShrinkFixed[T types.FixedSizeT](v *Vector, sels []int64) {
	vs := MustTCols[T](v)
	for i, sel := range sels {
		vs[i] = vs[sel]
	}
	v.Col = vs[:len(sels)]
	v.data = v.encodeColToByteSlice()
	v.Nsp = nulls.Filter(v.Nsp, sels)
}
func Shrink(v *Vector, sels []int64) {
	if v.IsScalar() {
		v.SetScalarLength(len(sels))
		return
	}

	switch v.Typ.Oid {
	case types.T_bool:
		ShrinkFixed[bool](v, sels)
	case types.T_int8:
		ShrinkFixed[int8](v, sels)
	case types.T_int16:
		ShrinkFixed[int16](v, sels)
	case types.T_int32:
		ShrinkFixed[int32](v, sels)
	case types.T_int64:
		ShrinkFixed[int64](v, sels)
	case types.T_uint8:
		ShrinkFixed[uint8](v, sels)
	case types.T_uint16:
		ShrinkFixed[uint16](v, sels)
	case types.T_uint32:
		ShrinkFixed[uint32](v, sels)
	case types.T_uint64:
		ShrinkFixed[uint64](v, sels)
	case types.T_float32:
		ShrinkFixed[float32](v, sels)
	case types.T_float64:
		ShrinkFixed[float64](v, sels)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		// XXX shrink varlena, but did not shrink area.  For our vector, this
		// may well be the right thing.  If want to shrink area as well, we
		// have to copy each varlena value and swizzle pointer.
		ShrinkFixed[types.Varlena](v, sels)
	case types.T_date:
		ShrinkFixed[types.Date](v, sels)
	case types.T_datetime:
		ShrinkFixed[types.Datetime](v, sels)
	case types.T_timestamp:
		ShrinkFixed[types.Timestamp](v, sels)
	case types.T_decimal64:
		ShrinkFixed[types.Decimal64](v, sels)
	case types.T_decimal128:
		ShrinkFixed[types.Decimal128](v, sels)
	case types.T_uuid:
		ShrinkFixed[types.Uuid](v, sels)
	case types.T_TS:
		ShrinkFixed[types.TS](v, sels)
	case types.T_Rowid:
		ShrinkFixed[types.Rowid](v, sels)
	case types.T_tuple:
		vs := v.Col.([][]interface{})
		for i, sel := range sels {
			vs[i] = vs[sel]
		}
		v.Col = vs[:len(sels)]
		v.Nsp = nulls.Filter(v.Nsp, sels)
	default:
		panic("vector shrink unknonw type")
	}
}

// Shuffle assumes we do not have dup in sels.
func ShuffleFixed[T types.FixedSizeT](v *Vector, sels []int64, m *mheap.Mheap) error {
	olddata := v.data
	ns := len(sels)
	vs := MustTCols[T](v)
	data, err := mheap.Alloc(m, int64(ns*v.GetType().TypeSize()))
	if err != nil {
		return err
	}
	ws := types.DecodeFixedSlice[T](data, v.GetType().TypeSize())
	v.Col = shuffle.FixedLengthShuffle(vs, ws, sels)
	v.data = types.EncodeFixedSlice(ws, v.GetType().TypeSize())
	v.Nsp = nulls.Filter(v.Nsp, sels)

	mheap.Free(m, olddata)
	return nil
}

func Shuffle(v *Vector, sels []int64, m *mheap.Mheap) error {
	if v.IsScalar() {
		v.SetScalarLength(len(sels))
		return nil
	}
	switch v.Typ.Oid {
	case types.T_bool:
		ShuffleFixed[bool](v, sels, m)
	case types.T_int8:
		ShuffleFixed[int8](v, sels, m)
	case types.T_int16:
		ShuffleFixed[int16](v, sels, m)
	case types.T_int32:
		ShuffleFixed[int32](v, sels, m)
	case types.T_int64:
		ShuffleFixed[int64](v, sels, m)
	case types.T_uint8:
		ShuffleFixed[uint8](v, sels, m)
	case types.T_uint16:
		ShuffleFixed[uint16](v, sels, m)
	case types.T_uint32:
		ShuffleFixed[uint32](v, sels, m)
	case types.T_uint64:
		ShuffleFixed[uint64](v, sels, m)
	case types.T_float32:
		ShuffleFixed[float32](v, sels, m)
	case types.T_float64:
		ShuffleFixed[float64](v, sels, m)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		ShuffleFixed[types.Varlena](v, sels, m)
	case types.T_date:
		ShuffleFixed[types.Date](v, sels, m)
	case types.T_datetime:
		ShuffleFixed[types.Datetime](v, sels, m)
	case types.T_timestamp:
		ShuffleFixed[types.Timestamp](v, sels, m)
	case types.T_decimal64:
		ShuffleFixed[types.Decimal64](v, sels, m)
	case types.T_decimal128:
		ShuffleFixed[types.Decimal128](v, sels, m)
	case types.T_uuid:
		ShuffleFixed[types.Uuid](v, sels, m)
	case types.T_TS:
		ShuffleFixed[types.TS](v, sels, m)
	case types.T_Rowid:
		ShuffleFixed[types.Rowid](v, sels, m)
	case types.T_tuple:
		vs := v.Col.([][]interface{})
		ws := make([][]interface{}, len(vs))
		v.Col = shuffle.TupleShuffle(vs, ws, sels)
		v.Nsp = nulls.Filter(v.Nsp, sels)
	default:
		panic(fmt.Sprintf("unexpect type %s for function vector.Shuffle", v.Typ))
	}
	return nil
}

func (v *Vector) Show() ([]byte, error) {
	// Write Typ
	var buf bytes.Buffer
	vtbs := types.EncodeType(&v.Typ)
	buf.Write(vtbs)

	// Write nspLen, nsp
	nb, err := v.Nsp.Show()
	if err != nil {
		return nil, err
	}

	lenNb := uint32(len(nb))
	buf.Write(types.EncodeUint32(&lenNb))
	if len(nb) > 0 {
		buf.Write(nb)
	}

	// Write colLen, col
	bs := v.encodeColToByteSlice()
	lenBs := uint32(len(bs))
	buf.Write(types.EncodeUint32(&lenBs))
	if len(bs) > 0 {
		buf.Write(bs)
	}

	// Write areaLen, area
	if len(v.area) == 0 {
		z := uint32(0)
		buf.Write(types.EncodeUint32(&z))
	} else {
		lenA := uint32(len(v.area))
		buf.Write(types.EncodeUint32(&lenA))
		buf.Write(v.area)
	}
	return buf.Bytes(), nil
}

func (v *Vector) Read(data []byte) error {
	typ := types.DecodeType(data[:types.TSize])
	data = data[types.TSize:]
	v.Typ = typ
	v.original = true

	// Read nspLen, nsp
	v.Nsp = &nulls.Nulls{}
	size := types.DecodeUint32(data)
	data = data[4:]
	if size > 0 {
		if err := v.Nsp.Read(data[:size]); err != nil {
			return err
		}
		data = data[size:]
	}

	// Read colLen, col,
	size = types.DecodeUint32(data)
	data = data[4:]
	if size > 0 {
		if v.GetType().IsTuple() {
			col := v.Col.([][]interface{})
			if err := types.Decode(data[:size], &col); err != nil {
				return err
			}
			v.Col = col
		} else {
			v.data = data[:size]
			v.setupColFromData(0, int(size/uint32(v.GetType().TypeSize())))
		}
		data = data[size:]
	} else {
		// This will give Col correct type.
		v.colFromData()
	}

	// Read areaLen and area
	size = types.DecodeUint32(data)
	if size != 0 {
		data = data[4:]
		v.area = data[:size]
	}
	return nil
}

// XXX Old Copy is FUBAR.
// Copy simply does v[vi] = w[wi]
func Copy(v, w *Vector, vi, wi int64, m *mheap.Mheap) error {
	if v.GetType().IsTuple() {
		// Not sure if Copy ever handle tuple
		panic("copy tuple vector.")
	} else if v.GetType().IsFixedLen() {
		tlen := int64(v.GetType().TypeSize())
		copy(v.data[vi*tlen:(vi+1)*tlen], w.data[wi*tlen:(wi+1)*tlen])
	} else {
		var err error
		vva := MustTCols[types.Varlena](v)
		wva := MustTCols[types.Varlena](w)
		if wva[wi].IsSmall() {
			vva[vi] = wva[wi]
		} else {
			bs := wva[wi].GetByteSlice(w.area)
			vva[vi], v.area, err = types.BuildVarlena(bs, v.area, m)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// XXX Old UnionOne is FUBAR
// It is simply append.   We do not go through appendOne interface because
// we don't want to horrible type switch.
func UnionOne(v, w *Vector, sel int64, m *mheap.Mheap) (err error) {
	if v.original {
		return moerr.NewInternalError("UnionOne cannot be performed on orig vector")
	}

	if err = v.extend(1, m); err != nil {
		return err
	}

	if v.GetType().IsTuple() {
		vs := v.Col.([][]interface{})
		ws := w.Col.([][]interface{})
		v.Col = append(vs, ws[sel])
		return nil
	}

	if nulls.Any(w.Nsp) && nulls.Contains(w.Nsp, uint64(sel)) {
		pos := uint64(v.Length() - 1)
		nulls.Add(v.Nsp, pos)
	} else if v.GetType().IsVarlen() {
		bs := w.GetBytes(sel)
		tgt := MustTCols[types.Varlena](v)
		nele := len(tgt)
		tgt[nele-1], v.area, err = types.BuildVarlena(bs, v.area, m)
		if err != nil {
			return err
		}
	} else {
		src := w.getRawValueAt(sel)
		tgt := v.getRawValueAt(-1)
		copy(tgt, src)
	}
	return nil
}

// XXX Old UnionNull is FUBAR
// func UnionNull(v, _ *Vector, m *mheap.Mheap) error
// It seems to do UnionOne(v, v, 0, m), only that if v is empty,
// append a zero value instead of v[0].   I don't know why this
// is called UnionNull -- it does not have much to do with Null.
//
// XXX Original code alloc or grow typesize * 8 bytes.  It is not
// clear people want to amortize alloc/grow, or it is a bug.
func UnionNull(v, _ *Vector, m *mheap.Mheap) error {
	if v.original {
		return moerr.NewInternalError("UnionNull cannot be performed on orig vector")
	}

	if v.Typ.IsTuple() {
		panic(moerr.NewInternalError("unionnull of tuple vector"))
	}

	if err := v.extend(1, m); err != nil {
		return err
	}

	// XXX old code actually copies, but it is a null, so what
	// is that good for.
	//
	// We don't care if v.GetType() is fixed len or not.  Since
	// v.area stays valid, a simple slice copy of Varlena works.
	// src := v.getRawValueAtOrZero(0)
	// tgt := v.getRawValueAt(-1)
	// copy(tgt, src)

	pos := uint64(v.Length() - 1)
	nulls.Add(v.Nsp, pos)
	return nil
}

// XXX Old Union is FUBAR
// Union is just append.
func Union(v, w *Vector, sels []int64, m *mheap.Mheap) (err error) {
	if v.original {
		return moerr.NewInternalError("Union cannot be performed on orig vector")
	}

	if err = v.extend(len(sels), m); err != nil {
		return err
	}

	if v.GetType().IsTuple() {
		panic("union called on tuple vector")
	} else if v.GetType().IsVarlen() {
		tgt := MustTCols[types.Varlena](v)
		next := len(tgt) - len(sels)
		for idx, sel := range sels {
			bs := w.GetBytes(sel)
			tgt[next+idx], v.area, err = types.BuildVarlena(bs, v.area, m)
			if err != nil {
				return err
			}
		}
	} else {
		next := -int64(len(sels))
		for idx, sel := range sels {
			src := w.getRawValueAt(sel)
			tgt := v.getRawValueAt(next + int64(idx))
			copy(tgt, src)
		}
	}
	return
}

// XXX Old UnionBatch is FUBAR.
func UnionBatch(v, w *Vector, offset int64, cnt int, flags []uint8, m *mheap.Mheap) (err error) {
	if v.original {
		return moerr.NewInternalError("UnionBatch cannot be performed on orig vector")
	}

	curIdx := v.Length()
	oldLen := uint64(curIdx)

	if err = v.extend(cnt, m); err != nil {
		return err
	}

	if v.GetType().IsTuple() {
		vs := v.Col.([][]interface{})
		ws := w.Col.([][]interface{})
		for i, flag := range flags {
			if flag > 0 {
				vs = append(vs, ws[int(offset)+i])
			}
		}
		v.Col = vs
	} else if v.GetType().IsVarlen() {
		tgt := MustTCols[types.Varlena](v)
		for idx, flg := range flags {
			if flg > 0 {
				bs := w.GetBytes(offset + int64(idx))
				tgt[curIdx], v.area, err = types.BuildVarlena(bs, v.area, m)
				curIdx += 1
			}
		}
	} else {
		for idx, flg := range flags {
			if flg > 0 {
				src := w.getRawValueAt(offset + int64(idx))
				tgt := v.getRawValueAt(int64(curIdx))
				copy(tgt, src)
				curIdx += 1
			}
		}
	}

	if nulls.Any(w.Nsp) {
		for idx, flg := range flags {
			if flg > 0 {
				if nulls.Contains(w.Nsp, uint64(offset)+uint64(idx)) {
					nulls.Add(v.Nsp, oldLen)
				}
				// Advance oldLen regardless if it is null
				oldLen += 1
			}
		}
	}

	return
}

// XXX Old Reset is FUBAR FUBAR.   I will put the code here just for fun.
func Reset(v *Vector) {
	/*
		switch v.Typ.Oid {
		case types.T_char, types.T_varchar, types.T_json, types.T_blob:
			v.Col.(*types.Bytes).Reset()
		default:
			// WTF is going on?
			*(*int)(unsafe.Pointer(uintptr((*(*emptyIntervade)(unsafe.Pointer(&v.Col))).word) + uintptr(strconv.IntSize>>3))) = 0
		}
	*/

	// XXX Reset does not do mem accounting?
	// I have no idea what is the purpose of Reset, so let me just Free it.
	// Maybe Reset want to keep v.data and v.area to save an allocation.
	// Let me do that ...
	v.setupColFromData(0, 0)
	v.area = v.area[:0]
	// XXX What about Nsp? Original code does not do anything to Nsp, which seems OK assuming
	// that will be set when we add data and we only test null within range of len(v.Col)
	// but who knows ...
}

// XXX What are these stuff, who use it?
func VecToString[T types.FixedSizeT](v *Vector) string {
	col := MustTCols[T](v)
	if len(col) == 1 {
		if nulls.Contains(v.Nsp, 0) {
			return "null"
		} else {
			return fmt.Sprintf("%v", col[0])
		}
	}
	// XXX Really?  What is this ...
	return fmt.Sprintf("%v-%s", v.Col, v.Nsp)
}

func (v *Vector) String() string {
	switch v.Typ.Oid {
	case types.T_bool:
		return VecToString[bool](v)
	case types.T_int8:
		return VecToString[int8](v)
	case types.T_int16:
		return VecToString[int16](v)
	case types.T_int32:
		return VecToString[int32](v)
	case types.T_int64:
		return VecToString[int64](v)
	case types.T_uint8:
		return VecToString[uint8](v)
	case types.T_uint16:
		return VecToString[uint16](v)
	case types.T_uint32:
		return VecToString[uint32](v)
	case types.T_uint64:
		return VecToString[uint64](v)
	case types.T_float32:
		return VecToString[float32](v)
	case types.T_float64:
		return VecToString[float64](v)
	case types.T_date:
		return VecToString[types.Date](v)
	case types.T_datetime:
		return VecToString[types.Datetime](v)
	case types.T_timestamp:
		return VecToString[types.Timestamp](v)
	case types.T_decimal64:
		return VecToString[types.Decimal64](v)
	case types.T_decimal128:
		return VecToString[types.Decimal128](v)
	case types.T_uuid:
		return VecToString[types.Uuid](v)
	case types.T_TS:
		return VecToString[types.TS](v)
	case types.T_Rowid:
		return VecToString[types.Rowid](v)
	case types.T_char, types.T_varchar, types.T_json, types.T_blob:
		col := MustStrCols(v)
		if len(col) == 1 {
			if nulls.Contains(v.Nsp, 0) {
				return "null"
			} else {
				return col[0]
			}
		}
		return fmt.Sprintf("%v-%s", v.Col, v.Nsp)

	default:
		panic("vec to string unknown types.")
	}
}
