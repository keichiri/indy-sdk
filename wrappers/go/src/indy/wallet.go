package indy

/*
#include <stdint.h>
#include <stdlib.h>
void indy_create_wallet_proxy(void *, int32_t, char *, char *, char *, char *, char *);
void indy_open_wallet_proxy(void *, int32_t, char *, char *, char *);
void indy_close_wallet_proxy(void *, int32_t, int32_t);
void indy_delete_wallet_proxy(void *, int32_t, char *, char *);
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"
)

func CreateWallet(poolName, name, xtype, config, credentials string) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_create_wallet")
	if err != nil {
		return err
	}

	var c_pool_name, c_name, c_xtype, c_config, c_credentials *C.char
	c_pool_name = C.CString(poolName)
	defer C.free(unsafe.Pointer(c_pool_name))
	c_name = C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	if xtype != "" {
		c_xtype = C.CString(xtype)
		defer C.free(unsafe.Pointer(c_xtype))
	}
	if config != "" {
		c_config = C.CString(config)
		defer C.free(unsafe.Pointer(c_config))
	}
	if credentials != "" {
		c_credentials = C.CString(credentials)
		defer C.free(unsafe.Pointer(c_credentials))
	}

	C.indy_create_wallet_proxy(pointer, C.int32_t(handle), c_pool_name, c_name,
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
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		panic("Received invalid handle in callback")
	}

	resCh <- code
}

type openWalletResult struct {
	code   int32
	handle int32
}

func OpenWallet(name string, runtimeConfig string, credentials string) (int32, error) {
	var c_name, c_runtime_config, c_credentials *C.char
	c_name = C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	if runtimeConfig != "" {
		c_runtime_config = C.CString(runtimeConfig)
		defer C.free(unsafe.Pointer(c_runtime_config))
	}
	if credentials != "" {
		c_credentials = C.CString(credentials)
		defer C.free(c_credentials)
	}

	pointer, handle, resCh, err := resolver.RegisterCall("indy_open_wallet")
	if err != nil {
		return -1, err
	}

	C.indy_open_wallet_proxy(pointer, C.int32_t(handle), c_name, c_runtime_config, c_credentials)

	_res := <-resCh
	res := _res.(*openWalletResult)

	if res.code != 0 {
		return -1, fmt.Errorf("IndySDK error code: %d", res.code)
	}

	return res.handle, nil
}

//export openWalletCallback
func openWalletCallback(commandHandle, code, walletHandle int32) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- &openWalletResult{
		code:   code,
		handle: walletHandle,
	}
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
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- code
}

func DeleteWallet(name string, credentials string) error {
	var c_name, c_credentials *C.char
	c_name = C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	if credentials != "" {
		c_credentials = C.CString(credentials)
		defer C.free(unsafe.Pointer(c_credentials))
	}

	pointer, handle, resCh, err := resolver.RegisterCall("indy_delete_wallet")
	if err != nil {
		return err
	}

	C.indy_delete_wallet_proxy(pointer, C.int32_t(handle), c_name, c_credentials)

	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("IndySDK error code: %d", res)
	}

	return nil
}

//export deleteWalletCallback
func deleteWalletCallback(commandHandle, code int32) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- code
}
