package indy

/*
#include <stdint.h>
void indy_create_wallet_proxy(void *, int32_t, char *, char *, char *, char *, char *);
void indy_open_wallet_proxy(void *, int32_t, char *, char *, char *);
void indy_close_wallet_proxy(void *, int32_t, int32_t);
void indy_delete_wallet_proxy(void *, int32_t, char *, char *);
*/
import "C"

import (
	"fmt"
	"log"
)

func CreateWallet(poolName, name, xtype, config, credentials string) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_create_wallet")
	if err != nil {
		return err
	}

	var c_xtype, c_config, c_credentials *C.char
	if xtype != "" {
		c_xtype = C.CString(xtype)
	}
	if config != "" {
		c_config = C.CString(config)
	}
	if credentials != "" {
		c_credentials = C.CString(credentials)
	}

	C.indy_create_wallet_proxy(pointer, C.int32_t(handle), C.CString(poolName), C.CString(name),
		c_xtype, c_config, c_credentials)

	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("IndySDK error code: %d", res)
	}

	return nil
}

//export createWalletCallback
func createWalletCallback(commandHandle, code int32) {
	log.Printf("In createWalletCallback. Command Handle: %d. Code: %d\n", commandHandle, code)
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		panic("Received invalid handle in callback")
	}

	ch <- code
	close(ch)
}

type openWalletResult struct {
	code   int32
	handle int32
}

func OpenWallet(name string, runtimeConfig string, credentials string) (int32, error) {
	var c_runtimeConfig *C.char
	if runtimeConfig != "" {
		c_runtimeConfig = C.CString(runtimeConfig)
	}

	var c_credentials *C.char
	if credentials != "" {
		c_credentials = C.CString(credentials)
	}

	pointer, handle, resCh, err := resolver.RegisterCall("indy_open_wallet")
	if err != nil {
		return -1, err
	}

	C.indy_open_wallet_proxy(pointer, C.int32_t(handle), C.CString(name), c_runtimeConfig, c_credentials)

	log.Printf("Gonna wait\n")
	_res := <-resCh
	res := _res.(*openWalletResult)
	log.Printf("Got result: %v\n", res)

	if res.code != 0 {
		return -1, fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.handle, nil
}

//export openWalletCallback
func openWalletCallback(commandHandle, code, walletHandle int32) {
	log.Printf("In open wallet callback\n")
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	res := &openWalletResult{
		code:   code,
		handle: walletHandle,
	}
	ch <- res
}

func CloseWallet(walletHandle int32) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_close_wallet")
	if err != nil {
		return err
	}

	C.indy_close_wallet_proxy(pointer, C.int32_t(handle), C.int32_t(walletHandle))

	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("IndySDK error code: %d", res)
	}

	return nil
}

//export closeWalletCallback
func closeWalletCallback(commandHandle, code int32) {
	log.Printf("In delete wallet callback\n")
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- code
}

func DeleteWallet(name string, credentials string) error {
	var c_credentials *C.char
	if credentials != "" {
		c_credentials = C.CString(credentials)
	}

	pointer, handle, resCh, err := resolver.RegisterCall("indy_delete_wallet")
	if err != nil {
		return err
	}

	C.indy_delete_wallet_proxy(pointer, C.int32_t(handle), C.CString(name), c_credentials)

	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("IndySDK error code: %d", res)
	}

	return nil
}

//export deleteWalletCallback
func deleteWalletCallback(commandHandle, code int32) {
	log.Printf("In delete wallet callback\n")
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- code
}
