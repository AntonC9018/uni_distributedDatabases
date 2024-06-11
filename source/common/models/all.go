package models

import "common/models/foaie"
import "common/models/client"

type Foaie = foaie.Foaie
type Client = client.Client

type AllTableModels struct {
    Client Client
    List Foaie
}

type ModelIndex int

const (
    ClientIndex ModelIndex = iota
    ListIndex ModelIndex = iota
)

const MaxIndex ModelIndex = ListIndex
const MinIndex ModelIndex = ClientIndex
const ModelCount int = (int)(MaxIndex - MinIndex) + 1

func (allModels *AllTableModels) Get(index ModelIndex) interface{} {
    switch index {
    case ClientIndex:
        var p = &allModels.Client
        return p
    case ListIndex:
        var p = &allModels.List
        return p
    default:
        panic("unreachable")
    }
}
