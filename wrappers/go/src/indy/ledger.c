#include <stdint.h>

extern void buildNymRequestCallback(int32_t, int32_t, char *);
extern void signAndSubmitRequestCallback(int32_t, int32_t, char *);
extern void buildGetNymRequestCallback(int32_t, int32_t, char *);
extern void submitRequestCallback(int32_t, int32_t, char *);


void indy_build_nym_request_proxy(void *f, int32_t handle, char *submitter, char *target,
    char *verkey, char *alias, char *role) {
    void (*func)(int32_t, char *, char *, char *, char *, char *, void *) = f;
    func(handle, submitter, target, verkey, alias, role, &buildNymRequestCallback);
}


void indy_sign_and_submit_request_proxy(void *f, int32_t handle, int32_t pool_handle,
    int32_t wallet_handle, char * submitter, char *request_json) {
    void (*func)(int32_t, int32_t, int32_t, char *, char *, void *) = f;
    func(handle, pool_handle, wallet_handle, submitter, request_json, &signAndSubmitRequestCallback);
}


void indy_build_get_nym_request_proxy(void *f, int32_t handle, char *submitter, char *target) {
    void (*func)(int32_t, char *, char *, void *) = f;
    func(handle, submitter, target, &buildGetNymRequestCallback);
}


void indy_submit_request_proxy(void *f, int32_t handle, int32_t pool_handle, char *request_json) {
    void (*func)(int32_t, int32_t, char *, void *) = f;
    func(handle, pool_handle, request_json, &submitRequestCallback);
}
