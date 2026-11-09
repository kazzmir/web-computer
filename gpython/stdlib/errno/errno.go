// Package errno exposes a common POSIX errno subset using Go's target-platform
// syscall values. Browser builds use Go's js/wasm errno table.
package errno

import (
	"github.com/kazzmir/web-computer/gpython/py"
	"syscall"
)

type constant struct {
	name string
	code syscall.Errno
}

var constants = []constant{
	{"EWOULDBLOCK", syscall.EWOULDBLOCK},
	{"EPERM", syscall.EPERM}, {"ENOENT", syscall.ENOENT}, {"ESRCH", syscall.ESRCH}, {"EINTR", syscall.EINTR},
	{"EIO", syscall.EIO}, {"ENXIO", syscall.ENXIO}, {"E2BIG", syscall.E2BIG}, {"ENOEXEC", syscall.ENOEXEC},
	{"EBADF", syscall.EBADF}, {"ECHILD", syscall.ECHILD}, {"EAGAIN", syscall.EAGAIN}, {"ENOMEM", syscall.ENOMEM},
	{"EACCES", syscall.EACCES}, {"EFAULT", syscall.EFAULT}, {"EBUSY", syscall.EBUSY}, {"EEXIST", syscall.EEXIST},
	{"EXDEV", syscall.EXDEV}, {"ENODEV", syscall.ENODEV}, {"ENOTDIR", syscall.ENOTDIR}, {"EISDIR", syscall.EISDIR},
	{"EINVAL", syscall.EINVAL}, {"ENFILE", syscall.ENFILE}, {"EMFILE", syscall.EMFILE}, {"ENOTTY", syscall.ENOTTY},
	{"EFBIG", syscall.EFBIG}, {"ENOSPC", syscall.ENOSPC}, {"ESPIPE", syscall.ESPIPE},
	{"EROFS", syscall.EROFS}, {"EMLINK", syscall.EMLINK}, {"EPIPE", syscall.EPIPE}, {"EDOM", syscall.EDOM},
	{"ERANGE", syscall.ERANGE}, {"EDEADLK", syscall.EDEADLK}, {"ENAMETOOLONG", syscall.ENAMETOOLONG},
	{"ENOLCK", syscall.ENOLCK}, {"ENOSYS", syscall.ENOSYS}, {"ENOTEMPTY", syscall.ENOTEMPTY}, {"ELOOP", syscall.ELOOP},
	{"ENOMSG", syscall.ENOMSG}, {"EIDRM", syscall.EIDRM}, {"EILSEQ", syscall.EILSEQ}, {"EOVERFLOW", syscall.EOVERFLOW},
	{"ENOTSOCK", syscall.ENOTSOCK}, {"EDESTADDRREQ", syscall.EDESTADDRREQ}, {"EMSGSIZE", syscall.EMSGSIZE},
	{"EPROTOTYPE", syscall.EPROTOTYPE}, {"ENOPROTOOPT", syscall.ENOPROTOOPT}, {"EPROTONOSUPPORT", syscall.EPROTONOSUPPORT},
	{"EOPNOTSUPP", syscall.EOPNOTSUPP}, {"EAFNOSUPPORT", syscall.EAFNOSUPPORT}, {"EADDRINUSE", syscall.EADDRINUSE},
	{"EADDRNOTAVAIL", syscall.EADDRNOTAVAIL}, {"ENETDOWN", syscall.ENETDOWN}, {"ENETUNREACH", syscall.ENETUNREACH},
	{"ENETRESET", syscall.ENETRESET}, {"ECONNABORTED", syscall.ECONNABORTED}, {"ECONNRESET", syscall.ECONNRESET},
	{"ENOBUFS", syscall.ENOBUFS}, {"EISCONN", syscall.EISCONN}, {"ENOTCONN", syscall.ENOTCONN},
	{"ETIMEDOUT", syscall.ETIMEDOUT}, {"ECONNREFUSED", syscall.ECONNREFUSED}, {"EHOSTUNREACH", syscall.EHOSTUNREACH},
	{"EALREADY", syscall.EALREADY}, {"EINPROGRESS", syscall.EINPROGRESS}, {"ECANCELED", syscall.ECANCELED},
	{"ENOTSUP", syscall.ENOTSUP},
}

func newErrorcode(self py.Object) (py.Object, error) {
	result := py.NewGenericDict()
	for _, entry := range constants {
		if _, err := result.M__setitem__(py.Int(entry.code), py.String(entry.name)); err != nil {
			return nil, err
		}
	}
	return result, nil
}
func init() {
	globals := py.NewStringDict()
	for _, entry := range constants {
		globals[entry.name] = py.Int(entry.code)
	}
	py.RegisterModule(&py.ModuleImpl{
		Info:    py.ModuleInfo{Name: "errno", Doc: "Common POSIX error constants and a reverse errorcode mapping; values follow Go's target platform."},
		Globals: globals,
		Methods: []*py.Method{py.MustNewMethod("_make_errorcode", newErrorcode, 0, "Create an independent reverse error-code dictionary.")},
		CodeSrc: "errorcode = _make_errorcode()\ndel _make_errorcode\n",
	})
}
