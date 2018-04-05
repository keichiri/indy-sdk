package indy

/*
#include <stdint.h>
#include <stdlib.h>
void indy_create_and_store_my_did_proxy(void *, int32_t, int32_t, char *);
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"
)

func CreateAndStoreMyDid(walletHandle int32, did string) (string, string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_create_and_store_my_did")
	if err != nil {
		return "", "", err
	}

	c_did := C.CString(did)
	defer C.free(unsafe.Pointer(c_did))

	C.indy_create_and_store_my_did_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle), c_did)
	_res := <-resCh
	res := _res.(*createAndStoreMyDidResult)
	if res.code != 0 {
		return "", "", fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.did, res.verkey, nil
}

type createAndStoreMyDidResult struct {
	code   int32
	did    string
	verkey string
}

//export createAndStoreMyDidCallback
func createAndStoreMyDidCallback(commandHandle, code int32, did *C.char, verkey *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &createAndStoreMyDidResult{
		code:   code,
		did:    C.GoString(did),
		verkey: C.GoString(verkey),
	}
}
