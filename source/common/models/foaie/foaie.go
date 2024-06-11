package foaie

import "common/iter"

type Foaie struct {
    Id int32
    Tip string
    Pret float64
    ProvidedTransport bool
    Hotel string
}

type Field int

const (
    IdField Field = iota
    TipField Field = iota
    PretField Field = iota
    ProvidedTransportField Field = iota
    HotelField Field = iota
)
const FirstField Field = IdField
const LastField Field = HotelField
const FieldCount int = (int)(HotelField) + 1

type ValueForEachField[T any] struct {
    Values [FieldCount]T
}
func (v *ValueForEachField[T]) Id() *T {
    return &v.Values[IdField]
}
func (v *ValueForEachField[T]) Tip() *T {
    return &v.Values[TipField]
}
func (v *ValueForEachField[T]) Pret() *T {
    return &v.Values[PretField]
}
func (v *ValueForEachField[T]) ProvidedTransport() *T {
    return &v.Values[ProvidedTransportField]
}
func (v *ValueForEachField[T]) Hotel() *T {
    return &v.Values[HotelField]
}

type IterValue[T any] struct {
    Key Field
    Value *T
}

func (v *ValueForEachField[T]) Iter() iter.Seq1[IterValue[T]] {
    return func(body func(IterValue[T]) bool) {
        for i := range v.Values {
            shouldKeepGoing := body(IterValue[T]{
                Key: (Field)(i),
                Value: &v.Values[i],
            })
            if !shouldKeepGoing {
                return
            }
        }
    }
}

func (v *ValueForEachField[T]) MaskedIter(mask FieldMask) iter.Seq1[IterValue[T]] {
    return func(body func(IterValue[T]) bool) {
        for i := range v.Values {
            val := IterValue[T]{
                Key: (Field)(i),
                Value: &v.Values[i],
            }
            if !mask.Get(val.Key) {
                continue
            }
            shouldKeepGoing := body(val)
            if !shouldKeepGoing {
                return
            }
        }
    }
}

type FieldMask struct {
    mask uint32
}

func (mask FieldMask) Get(bitIndex Field) bool {
    return (mask.mask & (1 << bitIndex)) != 0
}
func (mask *FieldMask) Set(bitIndex Field, set bool) {
    if set {
        mask.mask |= (1 << bitIndex)
    } else {
        mask.mask &= ^(1 << bitIndex)
    }
}

func (mask FieldMask) Id() bool {
    return mask.Get(IdField)
}
func (mask *FieldMask) SetId(set bool) {
    mask.Set(IdField, set)
}
func (mask FieldMask) Tip() bool {
    return mask.Get(TipField)
}
func (mask *FieldMask) SetTip(set bool) {
    mask.Set(TipField, set)
}
func (mask FieldMask) Pret() bool {
    return mask.Get(PretField)
}
func (mask *FieldMask) SetPret(set bool) {
    mask.Set(PretField, set)
}
func (mask FieldMask) ProvidedTransport() bool {
    return mask.Get(ProvidedTransportField)
}
func (mask *FieldMask) SetProvidedTransport(set bool) {
    mask.Set(ProvidedTransportField, set)
}
func (mask FieldMask) Hotel() bool {
    return mask.Get(HotelField)
}
func (mask *FieldMask) SetHotel(set bool) {
    mask.Set(HotelField, set)
}

func (mask *FieldMask) SetAll() {
    mask.mask = (uint32(1) << FieldCount) - 1;
}

func AllFieldMask() FieldMask {
    var ret FieldMask
    ret.SetAll()
    return ret
}

