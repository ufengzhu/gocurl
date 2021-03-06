package gocurl

// #cgo CFLAGS: -I/usr/include
// #cgo LDFLAGS: -lcurl
/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <curl/curl.h>

#include <sys/select.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>

// #define __GOCURL_DEBUG__

#ifdef __GOCURL_DEBUG__
#define dprintf(fmt, args...) do { \
	printf("%s():%d " fmt, __FUNCTION__, __LINE__, ##args); \
} while (0)
#else
#define dprintf(fmt, args...)
#endif

static CURLMcode curl_multi_setopt_long(CURLM *handle, CURLMoption option, long pointer) {
	return curl_multi_setopt(handle, option, pointer);
}

static CURLMcode curl_multi_setopt_off_t(CURLM *handle, CURLMoption option, off_t pointer) {
	return curl_multi_setopt(handle, option, pointer);
}

static CURLMcode curl_multi_setopt_str(CURLM *handle, CURLMoption option, const char *pointer) {
	return curl_multi_setopt(handle, option, pointer);
}

static CURLMcode curl_multi_setopt_ptr(CURLM *handle, CURLMoption option, void *pointer) {
	return curl_multi_setopt(handle, option, pointer);
}

static CURLMcode curl_multi_select(CURLM *handle, long ms, int *num) {
	struct timeval timeout;
	CURLMcode mc;
	int ret;

	fd_set fdread;
	fd_set fdwrite;
	fd_set fdexcep;
	int maxfd = -1;

	FD_ZERO(&fdread);
	FD_ZERO(&fdwrite);
	FD_ZERO(&fdexcep);

	timeout.tv_sec = ms / 1000;
	timeout.tv_usec = (ms % 1000) * 1000;

	mc = curl_multi_fdset(handle, &fdread, &fdwrite, &fdexcep, &maxfd);
	dprintf("mc = %d, maxfd = %d\n", mc, maxfd);
	if(mc != CURLM_OK) {
		dprintf("curl_multi_fdset failed, mc = %d\n", mc);
		return mc;
	}

	if (maxfd == -1) {
		dprintf("maxfd = -1\n");
		*num = 0;
		return CURLM_OK;
	}

	ret = select(maxfd+1, &fdread, &fdwrite, &fdexcep, &timeout);
	if (ret == -1) {
		return CURLM_BAD_SOCKET;
	} else if (ret == 0) {
		*num = 0;
		return CURLM_OK;
	}

	*num = ret;
	return CURLM_OK;
}
*/
import "C"
import "fmt"
import "unsafe"

type Multi struct {
	handle  unsafe.Pointer
	headers []unsafe.Pointer
}

type MultiError C.CURLMcode

var multiMap = make(map[unsafe.Pointer]*Multi)

func (mcode MultiError) Error() string {
	str := C.GoString(C.curl_multi_strerror(C.CURLMcode(mcode)))
	fmt.Printf("Multi error[%d]: %s\n", mcode, str)
	return fmt.Sprintf("Multi error[%d]: %s", mcode, str)
}

func codeToMError(mcode C.CURLMcode) error {
	if mcode != C.CURLM_OK {
		return MultiError(mcode)
	}

	return nil
}

func NewMulti() *Multi {
	handle := C.curl_multi_init()
	if handle == nil {
		return nil
	}

	multi := &Multi{}
	multi.handle = handle
	return multi
}

func (multi *Multi) Setopt(opt int, arg interface{}) error {
	if arg == nil {
		ret := C.curl_multi_setopt_ptr(multi.handle, C.CURLMoption(opt), unsafe.Pointer(nil))
		return codeToMError(ret)
	}

	switch {
	case opt >= OPTTYPE_OFF_T:
		val := C.off_t(0)
		switch arg.(type) {
		case int:
			val = C.off_t(arg.(int))
		case int64:
			val = C.off_t(arg.(int64))
		case uint64:
			val = C.off_t(arg.(uint64))
		default:
			fmt.Printf("Not implemented, %T, %v\n", arg, arg)
		}
		ret := C.curl_multi_setopt_off_t(multi.handle, C.CURLMoption(opt), val)
		return codeToMError(ret)

	case opt >= OPTTYPE_FUNCTIONPOINT:
		return fmt.Errorf("Not implemented: %d, %v", opt, arg)

	// case opt >= OPTTYPE_STRINGPOINT:
	case opt >= OPTTYPE_OBJECTPOINT:
		// OPT_URL
		switch arg.(type) {
		case string:
			cstr := C.CString(arg.(string))
			defer C.free(unsafe.Pointer(cstr))
			ret := C.curl_multi_setopt_str(multi.handle, C.CURLMoption(opt), cstr)
			return codeToMError(ret)

		case []string:
			// e.g. OPT_HTTPHEADER
			var list *C.struct_curl_slist = nil

			headers := arg.([]string)
			if len(headers) < 1 {
				break
			}
			for _, header := range headers {
				// fmt.Printf("Custom request header: %s\n", header)
				hdr := C.CString(header)
				defer C.free(unsafe.Pointer(hdr))
				// fmt.Printf("header: %T, %v\n", hdr, hdr)
				list = C.curl_slist_append(list, hdr)
			}
			ret := C.curl_multi_setopt_ptr(multi.handle, C.CURLOPT_HTTPHEADER, unsafe.Pointer(list))
			err := codeToMError(ret)
			if err != nil {
				return err
			}
			multi.headers = append(multi.headers, unsafe.Pointer(list))

		default:
			return fmt.Errorf("Not implemented: %d, %v", opt, arg)
		}

	case opt >= OPTTYPE_LONG:
		val := C.long(0)
		switch arg.(type) {
		case int:
			val = C.long(arg.(int))
		case bool:
			if arg.(bool) {
				val = 1
			}
		default:
			fmt.Printf("Not implemented, %T, %v\n", arg, arg)
		}
		ret := C.curl_multi_setopt_long(multi.handle, C.CURLMoption(opt), val)
		// fmt.Printf("curl_multi_setopt_long return %d\n", ret)
		return codeToMError(ret)

	default:
		fmt.Printf("Invalid option: %d\n", opt)
		return MultiError(E_UNKNOWN_OPTION)
	}

	return nil
}

func (multi *Multi) AddHandle(easy *Curl) error {
	ret := C.curl_multi_add_handle(multi.handle, easy.handle)
	return codeToMError(ret)
}

func (multi *Multi) RemoveHandle(easy *Curl) error {
	ret := C.curl_multi_remove_handle(multi.handle, easy.handle)
	return codeToMError(ret)
}

func (multi *Multi) Select(ms int) (int, error) {
	var num C.int = 0
	ret := C.curl_multi_select(multi.handle, C.long(ms), &num)
	return int(num), codeToMError(ret)
}

func (multi *Multi) Timeout() (int, error) {
	var ms C.long = 0
	ret := C.curl_multi_timeout(multi.handle, &ms)
	return int(ms), codeToMError(ret)
}

// timeout in millisecond
func (multi *Multi) Wait(timeout int) (int, error) {
	var num C.int = 0
	ret := C.curl_multi_wait(multi.handle, nil, 0, C.int(timeout), &num)
	return int(num), codeToMError(ret)
}

func (multi *Multi) Perform() (int, error) {
	var handles C.int = 0
	ret := C.curl_multi_perform(multi.handle, &handles)
	err := codeToMError(ret)
	return int(handles), err
}

func (multi *Multi) Cleanup() error {
	mcode := C.curl_multi_cleanup(multi.handle)
	return codeToMError(mcode)
}
