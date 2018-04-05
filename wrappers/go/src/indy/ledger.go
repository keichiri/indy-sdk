package indy

/*
#include <stdint.h>
#include <stdlib.h>
void indy_build_nym_request_proxy(void *, int32_t, char *, char *, char *, char *, char *);
void indy_sign_and_submit_request_proxy(void *, int32_t, int32_t, int32_t, char *, char *);
void indy_build_get_nym_request_proxy(void *, int32_t, char *, char *);
void indy_submit_request_proxy(void *, int32_t, int32_t, char *);
void indy_build_schema_request_proxy(void *, int32_t, char *, char *);
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"
)

func BuildNymRequest(submitterDID, targetDID, verKey, alias, role string) (string, error) {
	var c_submitter_did, c_target_did, c_verkey, c_alias, c_role *C.char
	c_submitter_did = C.CString(submitterDID)
	defer C.free(unsafe.Pointer(c_submitter_did))
	c_target_did = C.CString(targetDID)
	defer C.free(unsafe.Pointer(c_target_did))
	if verKey != "" {
		c_verkey = C.CString(verKey)
		defer C.free(unsafe.Pointer(c_verkey))
	}
	if alias != "" {
		c_alias = C.CString(alias)
		defer C.free(unsafe.Pointer(c_alias))
	}
	if role != "" {
		c_role = C.CString(role)
		defer C.free(unsafe.Pointer(c_role))
	}

	pointer, handle, resCh, err := resolver.RegisterCall("indy_build_nym_request")
	if err != nil {
		return "", err
	}

	C.indy_build_nym_request_proxy(pointer, C.int32_t(handle), c_submitter_did,
		c_target_did, c_verkey, c_alias, c_role)

	_res := <-resCh
	res := _res.(*buildNymRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.nymRequest, nil
}

type buildNymRequestResult struct {
	code       int32
	nymRequest string
}

//export buildNymRequestCallback
func buildNymRequestCallback(commandHandle, code int32, nymRequest *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &buildNymRequestResult{
		code:       code,
		nymRequest: C.GoString(nymRequest),
	}
}

///////

func SignAndSubmitRequest(poolHandle, walletHandle int32, submitterDID, request string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_sign_and_submit_request")
	if err != nil {
		return "", err
	}

	var c_submitter_did, c_request *C.char
	c_submitter_did = C.CString(submitterDID)
	defer C.free(unsafe.Pointer(c_submitter_did))
	c_request = C.CString(request)
	defer C.free(unsafe.Pointer(c_request))

	C.indy_sign_and_submit_request_proxy(pointer, C.int32_t(handle), C.int32_t(poolHandle),
		C.int32_t(walletHandle), c_submitter_did, c_request)

	_res := <-resCh
	res := _res.(*signAndSubmitRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.response, nil
}

type signAndSubmitRequestResult struct {
	code     int32
	response string
}

//export signAndSubmitRequestCallback
func signAndSubmitRequestCallback(commandHandle, code int32, response *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &signAndSubmitRequestResult{
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

	var c_submitter_did, c_target_did *C.char
	c_submitter_did = C.CString(submitterDID)
	defer C.free(unsafe.Pointer(c_submitter_did))
	c_target_did = C.CString(targetDID)
	defer C.free(unsafe.Pointer(c_target_did))

	C.indy_build_get_nym_request_proxy(pointer, C.int32_t(handle), c_submitter_did, c_target_did)

	_res := <-resCh
	res := _res.(*buildGetNymRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.getNymRequest, nil
}

type buildGetNymRequestResult struct {
	code          int32
	getNymRequest string
}

//export buildGetNymRequestCallback
func buildGetNymRequestCallback(commandHandle, code int32, getNymRequest *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &buildGetNymRequestResult{
		code:          code,
		getNymRequest: C.GoString(getNymRequest),
	}
}

///

func SubmitRequest(poolHandle int32, request string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_submit_request")
	if err != nil {
		return "", err
	}

	c_request := C.CString(request)
	defer C.free(unsafe.Pointer(c_request))

	C.indy_submit_request_proxy(pointer, C.int32_t(handle), C.int32_t(poolHandle), c_request)

	_res := <-resCh
	res := _res.(*submitRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.response, nil
}

type submitRequestResult struct {
	code     int32
	response string
}

//export submitRequestCallback
func submitRequestCallback(commandHandle, code int32, response *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &submitRequestResult{
		code:     code,
		response: C.GoString(response),
	}
}

////
func BuildSchemaRequest(submitterDID string, data string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_build_schema_request")
	if err != nil {
		return "", err
	}

	var c_submitter_did, c_data *C.char
	c_submitter_did = C.CString(submitterDID)
	defer C.free(unsafe.Pointer(c_submitter_did))
	c_data = C.CString(data)
	defer C.free(unsafe.Pointer(c_data))

	C.indy_build_schema_request_proxy(pointer, C.int32_t(handle), c_submitter_did, c_data)

	_res := <-resCh
	res := _res.(*buildSchemaRequestResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.schemaRequest, nil
}

type buildSchemaRequestResult struct {
	code          int32
	schemaRequest string
}

//export buildSchemaRequestCallback
func buildSchemaRequestCallback(commandHandle, code int32, schemaRequest *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callbac.\n")
		return
	}

	resCh <- &buildSchemaRequestResult{
		code:          code,
		schemaRequest: C.GoString(schemaRequest),
	}
}
