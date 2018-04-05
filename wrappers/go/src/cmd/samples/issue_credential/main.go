package main

import (
	"encoding/json"
	"indy"
	"log"
)

func main() {
	var poolName string = "pool"
	var walletName string = "wallet"

	// 1.
	log.Printf("\n1. Creates a new local pool ledger configuration that is used later " +
		"when connecting to ledger.\n")
	poolConfig := "{\"genesis_txn\": \"/home/vagrant/code/evernym/indy-sdk/cli/docker_pool_transactions_genesis\"}"
	err := indy.CreatePoolLedgerConfig(poolName, poolConfig)
	if err != nil {
		log.Fatalf("Error while creating pool ledger config: %s\n", err)
	}

	// 2.
	log.Printf("\n2. Open pool ledger and get handle from libindy.\n")
	poolHandle, err := indy.OpenPoolLedger(poolName, "")
	if err != nil {
		log.Fatalf("Error while opening pool ledger: %s\n", err)
	}
	log.Printf("Pool handle: %d\n", poolHandle)

	// 3.
	log.Printf("\n3. Creating new secure wallet.\n")
	err = indy.CreateWallet(poolName, walletName, "", "", "")
	if err != nil {
		log.Fatalf("Error while creating wallet: %s\n", err)
	}

	// 4.
	log.Printf("\n4. Open wallet and get handle from libindy\n")
	walletHandle, err := indy.OpenWallet(walletName, "", "")
	if err != nil {
		log.Fatalf("Error while opening wallet: %s\n", err)
	}
	log.Printf("Wallet handle: %d\n", walletHandle)

	// 5.
	log.Printf("\n5. Generating and storing Steward DID and Verkey\n")
	seedJSON := "{\"seed\": \"000000000000000000000000Steward1\"}"
	stewardDID, stewardVerkey, err := indy.CreateAndStoreMyDid(walletHandle, seedJSON)
	if err != nil {
		log.Fatalf("Failed to create and store DID: %s\n", err)
	}
	log.Printf("Steward DID: %s\n", stewardDID)
	log.Printf("Steward Verkey: %s\n", stewardVerkey)

	// 6.
	log.Printf("\n6. Generating and storing Trust Anchor DID and Verkey\n")
	trustAnchorDID, trustAnchorVerkey, err := indy.CreateAndStoreMyDid(walletHandle, "{}")
	if err != nil {
		log.Fatalf("Failed to create and store DID: %s\n", err)
	}
	log.Printf("Trust Anchor DID: %s\n", trustAnchorDID)
	log.Printf("Trust Anchor Verkey: %s\n", trustAnchorVerkey)

	// 7.
	log.Printf("\n7. Building NYM request to add Trust Anchor to the ledger\n")
	nymRequest, err := indy.BuildNymRequest(stewardDID, trustAnchorDID, trustAnchorVerkey, "", "TRUST_ANCHOR")
	if err != nil {
		log.Fatalf("Failed to build NYM request: %s", err)
	}
	log.Printf("NYM request: %s\n", nymRequest)

	// 8.
	log.Printf("\n8. Sending NYM request to the ledger\n")
	nymResponse, err := indy.SignAndSubmitRequest(poolHandle, walletHandle, stewardDID, nymRequest)
	if err != nil {
		log.Fatalf("Failed to submit NYM request: %s", err)
	}
	log.Printf("NYM response: %s\n", nymResponse)

	// 9.
	log.Printf("\n9. Build the SCHEMA request to add new schema to the ledger as a Steward\n")
	seqNo := 1
	schema := map[string]interface{}{
		"seqNo": seqNo,
		"dest":  stewardDID,
		"data": map[string]interface{}{
			"name":       "gvt",
			"version":    "1.0",
			"attr_names": []string{"age", "sex", "height", "name"},
		},
	}
	schemaData := schema["data"]
	log.Printf("Schema:\n%v\n", schemaData)
	schemaDataJSON, _ := json.Marshal(schemaData)
	schemaRequest, err := indy.BuildSchemaRequest(stewardDID, string(schemaDataJSON))
	if err != nil {
		log.Fatalf("Failed to build SCHEMA request: %s", err)
	}
	log.Printf("Schema request:\n%s\n", schemaRequest)

	// 10.
	log.Printf("\n10. Sending the SCHEMA request to the ledger\n")
	schemaResponse, err := indy.SignAndSubmitRequest(poolHandle, walletHandle, stewardDID, schemaRequest)
	if err != nil {
		log.Fatalf("Failed to submit schema to ledger: %s\n", err)
	}
	log.Printf("Schema response:\n%s\n", schemaResponse)

	// 11.
	log.Printf("\n11. Creating and storing claim definition using anoncreds as Trust Anchor, for the given schema\n")
	schemaJSON, _ := json.Marshal(schema)
	claimDefJson, err := indy.IssuerCreateAndStoreClaimDef(walletHandle, trustAnchorDID, string(schemaJSON), "CL", false)
	if err != nil {
		log.Printf("Failed to create claim definition: %s", err)
	}
	log.Printf("Claim definition: %s\n", claimDefJson)

	// 12.
	log.Printf("\n12. Creating Prover wallet and opening it to get the handle\n")
	proverDID := "VsKV7grR1BUE29mG2Fm2kX"
	proverWalletName := "proverWallet"
	err = indy.CreateWallet(poolName, proverWalletName, "", "", "")
	if err != nil {
		log.Fatalf("Failed to create wallet: %s", err)
	}
	proverWalletHandle, err := indy.OpenWallet(proverWalletName, "", "")
	if err != nil {
		log.Fatalf("Failed to open wallet: %s", err)
	}

	// 13.
	log.Printf("\n13. Creating Master Secret as Prover\n")
	masterSecretName := "master_secret"
	err = indy.ProverCreateMasterSecret(proverWalletHandle, masterSecretName)
	if err != nil {
		log.Fatalf("Failed to create master secret: %s", err)
	}

	// 14.
	log.Printf("\n14. Issuer (Trust Anchor) is creating a Claim Offer for Prover\n")
	claimOfferJson, err := indy.IssuerCreateClaimOffer(walletHandle, string(schemaJSON), trustAnchorDID, proverDID)
	if err != nil {
		log.Fatalf("failed to create claim offer: %s", err)
	}
	log.Printf("Claim Offer:%s\n%s", claimOfferJson)

	// 15.
	log.Printf("\n15. Prover creates Claim Request\n")
	claimReqJson, err := indy.ProverCreateAndStoreClaimReq(proverWalletHandle, proverDID, claimOfferJson, claimDefJson, masterSecretName)
	log.Printf("Claim Request:\n%s\n", claimReqJson)

	// 16.
	log.Printf("\n16. Issuer (Trust Anchor) creates Claim for Claim Request\n")
	claimData := map[string]interface{}{
		"sex":    []string{"male", "5944657099558967239210949258394887428692050081607692519917050011144233115103"},
		"name":   []string{"Alex", "1139481716457488690172217916278103335"},
		"height": []string{"175", "175"},
		"age":    []string{"28", "28"},
	}
	claimDataJson, _ := json.Marshal(claimData)
	_, claimJson, err := indy.IssuerCreateClaim(walletHandle, claimReqJson, string(claimDataJson), -1)
	if err != nil {
		log.Fatalf("failed to create claim: %s", err)
	}
	log.Printf("Claim:\n%s\n", claimJson)

	// 17.
	log.Printf("\n17. Prover processes and stores Claim\n")
	err = indy.ProverStoreClaim(proverWalletHandle, claimJson, "")
	if err != nil {
		log.Fatalf("Failed to store claim: %s", err)
	}

	// 18.
	log.Printf("\n18. Closing both wallets and pool\n")
	err = indy.CloseWallet(walletHandle)
	if err != nil {
		log.Fatalf("Failed to close wallet: %s", err)
	}
	err = indy.CloseWallet(proverWalletHandle)
	if err != nil {
		log.Fatalf("Failed to close wallet: %s", err)
	}
	err = indy.ClosePoolLedger(poolHandle)
	if err != nil {
		log.Fatalf("Failed to close pool ledger: %s", err)
	}

	// 19.
	log.Printf("\n19. Deleting created wallets\n")
	err = indy.DeleteWallet(walletName, "")
	if err != nil {
		log.Fatalf("Failed to delete wallet: %s", err)
	}
	err = indy.DeleteWallet(proverWalletName, "")
	if err != nil {
		log.Fatalf("Failed to delete wallet: %s", err)
	}

	// 20.
	log.Printf("\n20. Deleting pool ledger config\n")
	err = indy.DeletePoolLedgerConfig(poolName)
	if err != nil {
		log.Fatalf("Failed to delete pool ledger config: %s", err)
	}

	log.Printf("Successfully completed!")
}
