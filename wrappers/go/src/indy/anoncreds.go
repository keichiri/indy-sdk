package indy

/*
#include <stdint.h>
#include <stdlib.h>
#include <stdbool.h>
void indy_issuer_create_and_store_claim_def_proxy(void *, int32_t, int32_t, char *, char *, char *, bool);
void indy_prover_create_master_secret_proxy(void *, int32_t, int32_t, char *);
void indy_issuer_create_claim_offer_proxy(void *, int32_t, int32_t, char *, char *, char *);
void indy_prover_create_and_store_claim_req_proxy(void *, int32_t, int32_t, char *, char *, char *, char *);
void indy_issuer_create_claim_proxy(void *, int32_t, int32_t, char *, char *, int32_t);
void indy_prover_store_claim_proxy(void *, int32_t, int32_t, char *, char *);
*/
import "C"
import (
	"fmt"
	"log"
	"unsafe"
)

func IssuerCreateAndStoreClaimDef(walletHandle int32, issuerDID, schemaJSON,
	signatureType string, createNonRevoc bool) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_issuer_create_and_store_claim_def")
	if err != nil {
		return "", err
	}

	var c_issuer_did, c_schema_json, c_signature_type *C.char
	c_issuer_did = C.CString(issuerDID)
	defer C.free(unsafe.Pointer(c_issuer_did))
	c_schema_json = C.CString(schemaJSON)
	defer C.free(unsafe.Pointer(c_schema_json))
	if signatureType != "" {
		c_signature_type = C.CString(signatureType)
		defer C.free(unsafe.Pointer(c_signature_type))
	}

	C.indy_issuer_create_and_store_claim_def_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle),
		c_issuer_did, c_schema_json, c_signature_type, C.bool(createNonRevoc))
	_res := <-resCh
	res := _res.(*issuerCreateAndStoreClaimDefResult)
	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.claimDef, nil
}

type issuerCreateAndStoreClaimDefResult struct {
	code     int32
	claimDef string
}

//export issuerCreateAndStoreClaimDefCallback
func issuerCreateAndStoreClaimDefCallback(commandHandle, code int32, claimDef *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &issuerCreateAndStoreClaimDefResult{
		code:     code,
		claimDef: C.GoString(claimDef),
	}
}

///

func ProverCreateMasterSecret(walletHandle int32, masterSecretName string) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_prover_create_master_secret")
	if err != nil {
		return err
	}
	c_master_secret_name := C.CString(masterSecretName)
	defer C.free(unsafe.Pointer(c_master_secret_name))

	C.indy_prover_create_master_secret_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle), c_master_secret_name)

	_res := <-resCh
	res := _res.(int32)

	if res != 0 {
		return fmt.Errorf("Indy SDK error code: %d", res)
	}

	return nil
}

//export proverCreateMasterSecretCallback
func proverCreateMasterSecretCallback(commandHandle, code int32) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- code
}

///

func IssuerCreateClaimOffer(walletHandle int32, schema, issuerDID, proverDID string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_issuer_create_claim_offer")
	if err != nil {
		return "", err
	}

	var c_schema, c_issuer_did, c_prover_did *C.char
	c_schema = C.CString(schema)
	defer C.free(unsafe.Pointer(c_schema))
	c_issuer_did = C.CString(issuerDID)
	defer C.free(unsafe.Pointer(c_issuer_did))
	c_prover_did = C.CString(proverDID)
	defer C.free(unsafe.Pointer(c_prover_did))

	C.indy_issuer_create_claim_offer_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle),
		c_schema, c_issuer_did, c_prover_did)

	_res := <-resCh
	res := _res.(*issuerCreateClaimOfferResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.claimOffer, nil
}

type issuerCreateClaimOfferResult struct {
	code       int32
	claimOffer string
}

//export issuerCreateClaimOfferCallback
func issuerCreateClaimOfferCallback(commandHandle, code int32, claimOffer *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &issuerCreateClaimOfferResult{
		code:       code,
		claimOffer: C.GoString(claimOffer),
	}
}

/////

func ProverCreateAndStoreClaimReq(walletHandle int32, proverDID, claimOffer, claimDef,
	masterSecretName string) (string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_prover_create_and_store_claim_req")
	if err != nil {
		return "", err
	}
	var c_prover_did, c_claim_offer, c_claim_def, c_master_secret_name *C.char
	c_prover_did = C.CString(proverDID)
	defer C.free(unsafe.Pointer(c_prover_did))
	c_claim_offer = C.CString(claimOffer)
	defer C.free(unsafe.Pointer(c_claim_offer))
	c_claim_def = C.CString(claimDef)
	defer C.free(unsafe.Pointer(c_claim_def))
	c_master_secret_name = C.CString(masterSecretName)
	defer C.free(unsafe.Pointer(c_master_secret_name))

	C.indy_prover_create_and_store_claim_req_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle),
		c_prover_did, c_claim_offer, c_claim_def, c_master_secret_name)

	_res := <-resCh
	res := _res.(*proverCreateAndStoreClaimReqResult)

	if res.code != 0 {
		return "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.claimReq, nil
}

type proverCreateAndStoreClaimReqResult struct {
	code     int32
	claimReq string
}

//export proverCreateAndStoreClaimReqCallback
func proverCreateAndStoreClaimReqCallback(commandHandle, code int32, claimReq *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &proverCreateAndStoreClaimReqResult{
		code:     code,
		claimReq: C.GoString(claimReq),
	}
}

////
func IssuerCreateClaim(walletHandle int32, claimReq, claim string, userRevocIndex int) (string, string, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_issuer_create_claim")
	if err != nil {
		return "", "", err
	}

	var c_claim_req, c_claim *C.char
	c_claim_req = C.CString(claimReq)
	defer C.free(unsafe.Pointer(c_claim_req))
	c_claim = C.CString(claim)
	defer C.free(unsafe.Pointer(c_claim))

	C.indy_issuer_create_claim_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle),
		c_claim_req, c_claim, C.int32_t(userRevocIndex))

	_res := <-resCh
	res := _res.(*issuerCreateClaimResult)

	if res.code != 0 {
		return "", "", fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.revocUpdate, res.claim, nil
}

type issuerCreateClaimResult struct {
	code        int32
	revocUpdate string
	claim       string
}

//export issuerCreateClaimCallback
func issuerCreateClaimCallback(commandHandle, code int32, revocUpdate, claim *C.char) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &issuerCreateClaimResult{
		code:        code,
		revocUpdate: C.GoString(revocUpdate),
		claim:       C.GoString(claim),
	}
}

///

func ProverStoreClaim(walletHandle int32, claim, revocReg string) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_prover_store_claim")
	if err != nil {
		return err
	}
	var c_revoc_reg *C.char
	if revocReg != "" {
		c_revoc_reg = C.CString(revocReg)
	}

	c_claim := C.CString(claim)
	defer C.free(unsafe.Pointer(c_claim))

	C.indy_prover_store_claim_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle), c_claim, c_revoc_reg)

	_res := <-resCh
	res := _res.(int32)

	if res != 0 {
		return fmt.Errorf("Indy SDK error code: %d", res)
	}

	return nil
}

//export proverStoreClaimCallback
func proverStoreClaimCallback(commandHandle, code int32) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- code
}
