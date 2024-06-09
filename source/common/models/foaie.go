package models

import "common/iter"

type Foaie struct {
    Id int32
    Tip string
    Pret float64
    ProvidedTransport bool
    Hotel string
}

type FoaieField int

const (
    FoaieIdField FoaieField = iota
    FoaieTipField FoaieField = iota
    FoaiePretField FoaieField = iota
    FoaieProvidedTransportField FoaieField = iota
    FoaieHotelField FoaieField = iota
)
const FoaieFirstField FoaieField = FoaieIdField
const FoaieLastField FoaieField = FoaieHotelField
const FoaieFieldCount int = (int)(FoaieHotelField) + 1

type FoaieValueForEachField[T any] struct {
    Values [FoaieFieldCount]T
}
func (v *FoaieValueForEachField[T]) Id() *T {
    return &v.Values[FoaieIdField]
}
func (v *FoaieValueForEachField[T]) Tip() *T {
    return &v.Values[FoaieTipField]
}
func (v *FoaieValueForEachField[T]) Pret() *T {
    return &v.Values[FoaiePretField]
}
func (v *FoaieValueForEachField[T]) ProvidedTransport() *T {
    return &v.Values[FoaieProvidedTransportField]
}
func (v *FoaieValueForEachField[T]) Hotel() *T {
    return &v.Values[FoaieHotelField]
}

type ValueForEachIter[T any, Key any] struct {
}
type IterValue[T any] struct {
    Key FoaieField
    Value *T
}

func (v *FoaieValueForEachField[T]) Iter() iter.Seq1[IterValue[T]] {
    return func(body func(IterValue[T]) bool) {
        for i := range v.Values {
            shouldKeepGoing := body(IterValue[T]{
                Key: (FoaieField)(i),
                Value: &v.Values[i],
            })
            if !shouldKeepGoing {
                return
            }
        }
    }
}

func (v *FoaieValueForEachField[T]) MaskedIter(mask FoaieFieldMask) iter.Seq1[IterValue[T]] {
    return func(body func(IterValue[T]) bool) {
        for i := range v.Values {
            val := IterValue[T]{
                Key: (FoaieField)(i),
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

type FoaieFieldMask struct {
    mask uint32
}

func (mask FoaieFieldMask) Get(bitIndex FoaieField) bool {
    return (mask.mask & (1 << bitIndex)) != 0
}
func (mask *FoaieFieldMask) Set(bitIndex FoaieField, set bool) {
    if set {
        mask.mask |= (1 << bitIndex)
    } else {
        mask.mask &= ^(1 << bitIndex)
    }
}

func (mask FoaieFieldMask) Id() bool {
    return mask.Get(FoaieIdField)
}
func (mask *FoaieFieldMask) SetId(set bool) {
    mask.Set(FoaieIdField, set)
}
func (mask FoaieFieldMask) Tip() bool {
    return mask.Get(FoaieTipField)
}
func (mask *FoaieFieldMask) SetTip(set bool) {
    mask.Set(FoaieTipField, set)
}
func (mask FoaieFieldMask) Pret() bool {
    return mask.Get(FoaiePretField)
}
func (mask *FoaieFieldMask) SetPret(set bool) {
    mask.Set(FoaiePretField, set)
}
func (mask FoaieFieldMask) ProvidedTransport() bool {
    return mask.Get(FoaieProvidedTransportField)
}
func (mask *FoaieFieldMask) SetProvidedTransport(set bool) {
    mask.Set(FoaieProvidedTransportField, set)
}
func (mask FoaieFieldMask) Hotel() bool {
    return mask.Get(FoaieHotelField)
}
func (mask *FoaieFieldMask) SetHotel(set bool) {
    mask.Set(FoaieHotelField, set)
}

func (mask *FoaieFieldMask) SetAll() {
    mask.mask = (uint32(1) << FoaieFieldCount) - 1;
}

func FoaieAllFieldMask() FoaieFieldMask {
    var ret FoaieFieldMask
    ret.SetAll()
    return ret
}

