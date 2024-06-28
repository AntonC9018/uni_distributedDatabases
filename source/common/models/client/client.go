package client

import (
	"math/bits"
)

type IDType = int32

type Client struct {
    ID IDType
    Email string
    Nume string
    Prenume string
}

type Field int

const (
    IdField Field = iota
    EmailField Field = iota
    NumeField Field = iota
    PrenumeField Field = iota
)
const FirstField Field = IdField
const LastField Field = PrenumeField
const FieldCount int = (int)(LastField) + 1

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

func (mask *FieldMask) SetAll() {
    mask.mask = (uint32(1) << FieldCount) - 1;
}

func (mask FieldMask) Id() bool {
    return mask.Get(IdField)
}
func (mask *FieldMask) SetId(set bool) {
    mask.Set(IdField, set)
}
func (mask FieldMask) Email() bool {
    return mask.Get(EmailField)
}
func (mask *FieldMask) SetEmail(set bool) {
    mask.Set(EmailField, set)
}
func (mask FieldMask) Nume() bool {
    return mask.Get(NumeField)
}
func (mask *FieldMask) SetNume(set bool) {
    mask.Set(NumeField, set)
}
func (mask FieldMask) Prenume() bool {
    return mask.Get(PrenumeField)
}
func (mask *FieldMask) SetPrenume(set bool) {
    mask.Set(PrenumeField, set)
}

func (mask FieldMask) Empty() bool {
    return mask.mask == 0
}
func (mask FieldMask) Count() int {
    return bits.OnesCount32(mask.mask)
}
func (mask FieldMask) Difference(other FieldMask) FieldMask {
    return FieldMask{mask: mask.mask ^ other.mask}
}
