package credentials

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cryptProtectUIForbidden  = 0x1
	cryptProtectLocalMachine = 0x4
)

// protectFlags uses machine scope (CRYPTPROTECT_LOCAL_MACHINE) so that the
// credential can be written by the installer (running as an administrator)
// and read by the service (running as LocalSystem), which use different
// user DPAPI keys.
const protectFlags = cryptProtectUIForbidden | cryptProtectLocalMachine

var (
	crypt32                = windows.NewLazySystemDLL("crypt32")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func dpapiProtect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot protect empty data with dpapi")
	}
	var in, out dataBlob
	in.cbData = uint32(len(data))
	in.pbData = (*byte)(unsafe.Pointer(&data[0]))

	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		uintptr(protectFlags),
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, fmt.Errorf("failed to protect data with dpapi: %w", err)
	}
	protected := make([]byte, out.cbData)
	copy(protected, unsafe.Slice(out.pbData, int(out.cbData)))
	windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.pbData))))
	return protected, nil
}

func dpapiUnprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot unprotect empty data with dpapi")
	}
	var in, out dataBlob
	in.cbData = uint32(len(data))
	in.pbData = (*byte)(unsafe.Pointer(&data[0]))

	r, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectUIForbidden),
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, fmt.Errorf("failed to unprotect data with dpapi: %w", err)
	}
	unprotected := make([]byte, out.cbData)
	copy(unprotected, unsafe.Slice(out.pbData, int(out.cbData)))
	windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.pbData))))
	return unprotected, nil
}
