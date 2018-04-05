package indy

/*
#include <stdint.h>
#include <stdlib.h>
void indy_build_nym_request_proxy(void *, int32_t, char *, char *, char *, char *, char *);
void indy_sign_and_submit_request_proxy(void *, int32_t, int32_t, int32_t, char *, char *);
void indy_build_get_nym_request_proxy(void *, int32_t, char *, char *);
void indy_submit_request_proxy(void *, int32_t, int32_t, char *);
*/
import "C"

import (
	"fmt"
	"log"
)

func BuildNymRequest(submitterDID, targetDID, verKey, alias, role string) (string, error) {
	var c_verkey, c_alias, c_role *C.char
	if verKey != "" {
		c_verkey = C.CString(verKey)
		// defer C.free(c_verkey)
	}
	if alias != "" {
		c_alias = C.CString(alias)
		// defer C.free(c_alias)
	}
	if role != "" {
		c_role = C.CString(role)
		// defer C.free(c_role)
	}

	pointer, handle, resCh, err := resolver.RegisterCall("indy_build_nym_request")
	if err != nil {
		return "", err
	}

	C.indy_build_nym_request_proxy(pointer, C.int32_t(handle), C.CString(submitterDID),
		C.CString(targetDID), c_verkey, c_alias, c_role)

	_res := <-resCh
	res := _res.(*buildNymRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.nymRequest, nil
}

type buildNymRequestResult struct {
	code       int32
	nymRequest string
}

//export buildNymRequestCallback
func buildNymRequestCallback(commandHandle, code int32, nymRequest *C.char) {
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- &buildNymRequestResult{
		code:       code,
		nymRequest: C.GoString(nymRequest),
	}
}

///////

func SignAndSubmitRequest(poolHandle, walletHandle int32, submitterDID, requestJSON string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_sign_and_submit_request")
	if err != nil {
		return "", err
	}

	C.indy_sign_and_submit_request_proxy(pointer, C.int32_t(handle), C.int32_t(poolHandle),
		C.int32_t(walletHandle), C.CString(submitterDID), C.CString(requestJSON))

	_res := <-resCh
	res := _res.(*signAndSubmitRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.response, nil
}

type signAndSubmitRequestResult struct {
	code     int32
	response string
}

//export signAndSubmitRequestCallback
func signAndSubmitRequestCallback(commandHandle, code int32, response *C.char) {
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- &signAndSubmitRequestResult{
		code:     code,
		response: C.GoString(response),
	}
}

///////

func BuildGetNymRequest(submitterDID, targetDID string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_build_get_nym_request")
	if err != nil {
		return "", err
	}

	C.indy_build_get_nym_request_proxy(pointer, C.int32_t(handle), C.CString(submitterDID), C.CString(targetDID))

	_res := <-resCh
	res := _res.(*buildGetNymRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.getNymRequest, nil
}

type buildGetNymRequestResult struct {
	code          int32
	getNymRequest string
}

//export buildGetNymRequestCallback
func buildGetNymRequestCallback(commandHandle, code int32, getNymRequest *C.char) {
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- &buildGetNymRequestResult{
		code:          code,
		getNymRequest: C.GoString(getNymRequest),
	}
}

///

func SubmitRequest(poolHandle int32, requestJSON string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_submit_request")
	if err != nil {
		return "", err
	}

	C.indy_submit_request_proxy(pointer, C.int32_t(handle), C.int32_t(poolHandle), C.CString(requestJSON))

	_res := <-resCh
	res := _res.(*submitRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.response, nil
}

type submitRequestResult struct {
	code     int32
	response string
}

//export submitRequestCallback
func submitRequestCallback(commandHandle, code int32, response *C.char) {
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- &submitRequestResult{
		code:     code,
		response: C.GoString(response),
	}
}
