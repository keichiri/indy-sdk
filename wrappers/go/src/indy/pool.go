package indy

/*
#cgo LDFLAGS: -ldl
#include <stdint.h>
#include <stdlib.h>
void indy_open_pool_ledger_proxy(void *, int32_t, char *, char *);
void indy_create_pool_ledger_config_proxy(void *, int32_t, char *, char *);
void indy_close_pool_ledger_proxy(void *, int32_t, int32_t);
void indy_delete_pool_ledger_config_proxy(void *, int32_t, char *);
*/
import "C"

import (
	"fmt"
	"log"
)

func CreatePoolLedgerConfig(configName, config string) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_create_pool_ledger_config")
	if err != nil {
		return err
	}

	var c_config_name, c_config *C.char
	c_config_name = C.CString(configName)
	defer C.free(c_config_name)

	if config != "" {
		c_config = C.CString(config)
		defer C.free(c_config)
	}

	C.indy_create_pool_ledger_config_proxy(pointer, C.int32_t(handle), c_config_name, c_config)

	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("IndySDK error code: %d", res)
	}

	return nil
}

//export createPoolLedgerConfigCallback
func createPoolLedgerConfigCallback(commandHandle, code int32) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		panic("Received invalid handle in callback")
	}

	resCh <- code
}

type openPoolLedgerResult struct {
	code       int32
	poolHandle int32
}

func OpenPoolLedger(configName, config string) (int32, error) {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_open_pool_ledger")
	if err != nil {
		return -1, err
	}

	var c_config_name, c_config *C.char
	c_config_name = C.CString(configName)
	defer C.free(c_config_name)

	if config != "" {
		c_config = C.CString(config)
		defer C.free(c_config)
	}

	C.indy_open_pool_ledger_proxy(pointer, C.int32_t(handle), c_config_name, c_config)

	_res := <-resCh
	res := _res.(*openPoolLedgerResult)
	if res.code != 0 {
		return -1, fmt.Errorf("Indy SDK error code: %d", res.code)
	}

	return res.poolHandle, nil
}

//export openPoolLedgerCallback
func openPoolLedgerCallback(commandHandle, code, poolHandle int32) {
	resCh, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	resCh <- openPoolLedgerResult{code, poolHandle}
}

// func RefreshPoolLedger(poolHandle int32) error {
// 	fp, handle, resCh, err := resolver.RegisterCall("indy_refresh_pool_ledger")
// 	if err != nil {
// 		return err
// 	}
//
// 	C.indy_refresh_pool_ledger_proxy(fp, C.int32_t(handle), C.int32_t(poolHandle))
//
// 	_res := <-resCh
// 	res := _res.(int32)
// 	if res.errorCode != 0 {
// 		return fmt.Errorf("Indy SDK error code: %d", res.errCode)
// 	}
//
// 	return nil
// }

// close pool ledger
func ClosePoolLedger(poolHandle int32) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_close_pool_ledger")
	if err != nil {
		return err
	}

	C.indy_close_pool_ledger_proxy(pointer, C.int32_t(handle), C.int32_t(poolHandle))
	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("Indy SDK error code: %d", res)
	}

	return nil
}

//export closePoolLedgerCallback
func closePoolLedgerCallback(commandHandle int32, code int32) {
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- code
}

// indy_delete_pool_ledger_config
func DeletePoolLedgerConfig(poolName string) error {
	pointer, handle, resCh, err := resolver.RegisterCall("indy_delete_pool_ledger_config")
	if err != nil {
		return err
	}

	C.indy_delete_pool_ledger_config_proxy(pointer, C.int32_t(handle), C.CString(poolName))
	_res := <-resCh
	res := _res.(int32)
	if res != 0 {
		return fmt.Errorf("Indy SDK error code: %d", res)
	}

	return nil
}

//export deletePoolLedgerConfigCallback
func deletePoolLedgerConfigCallback(commandHandle int32, code int32) {
	ch, err := resolver.DeregisterCall(commandHandle)
	if err != nil {
		log.Printf("ERROR: invalid handle in callback.\n")
		return
	}

	ch <- code
}
