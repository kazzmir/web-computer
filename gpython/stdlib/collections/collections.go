// Package collections currently provides a UserList compatibility subset.
package collections

import (
	_ "embed"
	"github.com/kazzmir/web-computer/gpython/py"
)

//go:embed userlist.py
var source string

func init() {
	py.RegisterModule(&py.ModuleImpl{Info: py.ModuleInfo{Name: "collections", Doc: "UserList sequence wrapper subset"}, CodeSrc: source})
}
