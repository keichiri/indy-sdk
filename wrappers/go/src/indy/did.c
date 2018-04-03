#include <stdint.h>


extern void createAndStoreMyDidCallback(int32_t, int32_t, char *, char *);


void indy_create_and_store_my_did_proxy(void *f, int32_t handle, char *did) {
    void (*func)(int32_t, char *, void *) = f;
    func(handle, did, &createAndStoreMyDidCallback);
}
