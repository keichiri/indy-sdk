#include <stdint.h>

extern void createWalletCallback(int32_t, int32_t);
extern void openWalletCallback(int32_t, int32_t, int32_t);
extern void deleteWalletCallback(int32_t, int32_t);
extern void closeWalletCallback(int32_t, int32_t);


void indy_create_wallet_proxy(void *f, int32_t handle, char *pool_name, char *name, char *xtype, char *config, char *credentials) {
    void (*func)(int32_t, char *, char *, char *, char *, char *, void *) = f;
    func(handle, pool_name, name, xtype, config, credentials, &createWalletCallback);
}


void indy_open_wallet_proxy(void *f, int32_t handle, char *name, char *runtime_config, char *credentials) {
    void (*func)(int32_t, char *, char *, char *, void *) = f;
    func(handle, name, runtime_config, credentials, &openWalletCallback);
}


void indy_delete_wallet_proxy(void *f, int32_t handle, char *name, char *credentials) {
    void (*func)(int32_t, char *, char *, void *) = f;
    func(handle, name, credentials, &deleteWalletCallback);
}


void indy_close_wallet_proxy(void *f, int32_t handle, int32_t wallet_handle) {
    void (*func)(int32_t, int32_t, void *) = f;
    func(handle, wallet_handle, &deleteWalletCallback);
}
