package main

type Client struct {
    Id int32
    Email string
    Nume string
    Prenume string
}

type ListType string

type List struct {
    Id int32
    Tip ListType
    Pret float64
    ProvidedTransport bool
    Hotel string
}

type AllTableModels struct {
    Client Client
    List List
}

type ModelIndex int

const (
    ClientIndex ModelIndex = iota
    ListIndex ModelIndex = iota
)

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