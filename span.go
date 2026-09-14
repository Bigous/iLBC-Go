package ilbc

// span preserves the base of an RFC pointer, including when a filter walks
// backwards into its history. Dereferences retain Go's bounds checks.
type span[T any] struct {
	data []T
	off  int
}

func (p span[T]) add(n int) span[T] { p.off += n; return p }
func (p span[T]) at(n int) *T       { return &p.data[p.off+n] }
func (p span[T]) slice(n int) []T   { return p.data[p.off : p.off+n] }
