package models

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

