package iter

type Seq2[T1 any, T2 any] func(func(T1, T2) bool)
type Seq1[T1 any] func(func(T1) bool)
