#include <stdint.h>
#include <stdbool.h>


extern void issuerCreateAndStoreClaimDefCallback(int32_t, int32_t, char *);
extern void proverCreateMasterSecretCallback(int32_t, int32_t);
extern void issuerCreateClaimOfferCallback(int32_t, int32_t, char *);
extern void proverCreateAndStoreClaimReqCallback(int32_t, int32_t, char *);
extern void issuerCreateClaimCallback(int32_t, int32_t, char *, char *);
extern void proverStoreClaimCallback(int32_t, int32_t);


void indy_issuer_create_and_store_claim_def_proxy(void *f, int32_t handle, int32_t wallet_handle,
    char *issuer_did, char *schema_json, char *signature_type, bool create_non_revoc) {
    void (*func)(int32_t, int32_t, char *, char *, char *, bool, void *) = f;
    func(handle, wallet_handle, issuer_did, schema_json, signature_type, create_non_revoc, &issuerCreateAndStoreClaimDefCallback);
}


void indy_prover_create_master_secret_proxy(void *f, int32_t handle, int32_t wallet_handle, char *master_secret_name) {
    void (*func)(int32_t, int32_t, char *, void *) = f;
    func(handle, wallet_handle, master_secret_name, &proverCreateMasterSecretCallback);
}


void indy_issuer_create_claim_offer_proxy(void *f, int32_t handle, int32_t wallet_handle,
    char *schema, char *issuer_did, char *prover_did) {
    void (*func)(int32_t, int32_t, char *, char *, char *, void *) = f;
    func(handle, wallet_handle, schema, issuer_did, prover_did, &issuerCreateClaimOfferCallback);
}


void indy_prover_create_and_store_claim_req_proxy(void *f, int32_t handle, int32_t wallet_handle,
    char *prover_did, char *claim_offer, char *claim_def, char *master_secret_name) {
    void (*func)(int32_t, int32_t, char *, char *, char *, char *, void *) = f;
    func(handle, wallet_handle, prover_did, claim_offer, claim_def, master_secret_name, &proverCreateAndStoreClaimReqCallback);
}


void indy_issuer_create_claim_proxy(void *f, int32_t handle, int32_t wallet_handle,
    char *claim_req, char *claim, int32_t user_revoc_index) {
    void (*func)(int32_t, int32_t, char *, char *, int32_t, void *) = f;
    func(handle, wallet_handle, claim_req, claim, user_revoc_index, &issuerCreateClaimCallback);
}


void indy_prover_store_claim_proxy(void *f, int32_t handle, int32_t wallet_handle, char *claim, char *revoc_reg) {
    void (*func)(int32_t, int32_t, char *, char *, void *) = f;
    func(handle, wallet_handle, claim, revoc_reg, &proverStoreClaimCallback);
}
