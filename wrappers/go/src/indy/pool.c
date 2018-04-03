#include <stdint.h>

extern void openPoolLedgerCallback(int32_t, int32_t, int32_t);
extern void createPoolLedgerConfigCallback(int32_t, int32_t);
extern void closePoolLedgerCallback(int32_t, int32_t);
extern void deletePoolLedgerConfigCallback(int32_t, char *);


void indy_open_pool_ledger_proxy(void *f, int32_t handle, char *config_name, char *config) {
    void (*func)(int32_t, char *, char *, void (*)(int32_t, int32_t, int32_t)) = f;
    func(handle, config_name, config, &openPoolLedgerCallback);
}


void indy_create_pool_ledger_config_proxy(void *f, int32_t handle, char *config_name, char *config) {
    void (*func)(int32_t, char *, char *, void (*)(int32_t, int32_t)) = f;
    func(handle, config_name, config, &createPoolLedgerConfigCallback);
}


void indy_close_pool_ledger_proxy(void *f, int32_t handle, int32_t pool_handle) {
    void (*func)(int32_t, int32_t, void (*)(int32_t, int32_t)) = f;
    func(handle, pool_handle, &closePoolLedgerCallback);
}


void indy_delete_pool_ledger_config_proxy(void *f, int32_t handle, char *pool_name) {
    void (*func)(int32_t, char *, void (*)(int32_t, int32_t)) = f;
    func(handle, pool_name, &closePoolLedgerCallback);
}
