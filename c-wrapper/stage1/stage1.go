package main

// #import "helper.h"
import "C"

import (
	"unsafe"

	"github.com/OpenICP-BR/libICP"
	pointer "github.com/mattn/go-pointer"
)

func new_vec_ptr(src []interface{}) []unsafe.Pointer {
	ans := make([]unsafe.Pointer, len(src))
	for i, v := range src {
		ans[i] = unsafe.Pointer(&v)
	}
	return ans
}

//export Version
func Version() *C.char {
	return C.CString(libICP.Version())
}

//export CodedErrorGetErrorStr
func CodedErrorGetErrorStr(ptr unsafe.Pointer) *C.char {
	errc := pointer.Restore(ptr).(libICP.CodedError)
	return C.CString(errc.CodeString())
}

//export CodedErrorGetErrorInt
func CodedErrorGetErrorInt(ptr unsafe.Pointer) C.int {
	errc := pointer.Restore(ptr).(libICP.CodedError)
	return C.int(errc.Code())
}

//export CertSubject
func CertSubject(ptr unsafe.Pointer) *C.char {
	cert := pointer.Restore(ptr).(*libICP.Certificate)
	return C.CString(cert.Subject)
}

//export CertSubjectMap
func CertSubjectMap(cert_ptr unsafe.Pointer, output *C.name_entry) {
	cert := pointer.Restore(cert_ptr).(*libICP.Certificate)
	i := 0
	for key, val := range cert.SubjectMap {
		C.set_name_entry_key(output, C.int(i), C.CString(key))
		C.set_name_entry_val(output, C.int(i), C.CString(val))
		i++
	}
}

//export CertIssuerMap
func CertIssuerMap(cert_ptr unsafe.Pointer, output *C.name_entry) {
	cert := pointer.Restore(cert_ptr).(*libICP.Certificate)
	i := 0
	for key, val := range cert.IssuerMap {
		C.set_name_entry_key(output, C.int(i), C.CString(key))
		C.set_name_entry_val(output, C.int(i), C.CString(val))
		i++
	}
}

//export CertIssuer
func CertIssuer(ptr unsafe.Pointer) *C.char {
	cert := pointer.Restore(ptr).(*libICP.Certificate)
	return C.CString(cert.Issuer)
}

//export ErrorStr
func ErrorStr(ptr unsafe.Pointer) *C.char {
	err := pointer.Restore(ptr).(error)
	return C.CString(err.Error())
}

//export NewCertificateFromFile
func NewCertificateFromFile(path_c *C.char, certs_ptr *unsafe.Pointer, errcs_ptr *unsafe.Pointer, buf_size int) C.int {
	path := C.GoString(path_c)

	ans_certs, ans_errs := libICP.NewCertificateFromFile(path)

	for i := range ans_certs {
		if i >= buf_size {
			break
		}
		ptr := pointer.Save(&ans_certs[i])
		C.set_void_vet_ptr(certs_ptr, C.int(i), ptr)
	}
	for i := range ans_errs {
		if i >= buf_size {
			break
		}
		ptr := pointer.Save(ans_errs[i])
		C.set_void_vet_ptr(errcs_ptr, C.int(i), ptr)
	}

	return C.int(len(ans_errs))
}

func main() {}