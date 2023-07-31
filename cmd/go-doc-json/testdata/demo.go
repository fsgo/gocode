package testdata

import (
	"context"
	"net"
	"sync"
)

// CType1 类型 1
type CType1 int

const (
	// C1 c1 的文档
	C1 CType1 = iota
	C2        // c2 的文档
	c3        // c3 是私有的
)

// User doc for user
type User struct {
	id   int
	Name string
}

// Hello doc for Hello
//
// line3
func (u *User) Hello() {}

func (u *User) Say(msg string) error {
	return nil
}

// Print1 doc for Print1
func Print1() {}

func Print2(msg int, add net.Addr) {}

type MemStats struct {
	BySize [61]struct {
		// Size is the maximum byte size of an object in this
		ID string
	}
}

type Cache[K any, V any] struct {
}

func (c *Cache[K, V]) Register() {}

func Getaddrinfo(hostname, servname *byte, hints *User, res **User) (int, error) {
	return 0, nil
}

type Func1 func(ctx context.Context)

type Interface struct {
	check     *User    // for error reporting; nil once type set is computed
	methods   []*Func1 // ordered list of explicitly declared methods
	embeddeds []User   // ordered list of explicitly embedded elements
	embedPos  *[]User  // positions of embedded elements; or nil (for error messages) - use pointer to save space
	implicit  bool     // interface is wrapper for type set literal (non-interface T, ~T, or A|B)
	complete  bool     // indicates that obj, methods, and embeddeds are set and type set can be computed
	tset      *User    // type set described by this interface, computed lazily
}

type (
	IUnknown struct {
		// RawVTable *interface{}
		RawVTable *interface{} // 注意，这个不是 *any
	}

	LChannel chan User
)

type XXX_InternalExtensions struct {
	p *struct {
  		mu           sync.Mutex
   		extensionMap map[int32]User
   	}
}


