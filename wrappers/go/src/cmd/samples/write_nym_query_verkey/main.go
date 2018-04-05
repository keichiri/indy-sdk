package main

import (
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
	log.Printf("\n9. Generating and storing DID and Verkey representing a Client that wants " +
		"to obtain Trust Anchor's Verkey\n")
	clientDID, _, err := indy.CreateAndStoreMyDid(walletHandle, "{}")
	if err != nil {
		log.Fatalf("Failed to create and store DID: %s\n", err)
	}

	// 10.
	log.Printf("\n10. Building the GET_NYM request to query Trust Anchor's Verkey from ledger\n")
	getNymRequest, err := indy.BuildGetNymRequest(clientDID, trustAnchorDID)
	if err != nil {
		log.Printf("Failed to build GET_NYM request: %s", err)
	}
	log.Printf("GET_NYM request: %s\n", getNymRequest)

	// 11.
	log.Printf("\n11. Sending the GET_NYM request to the lerger\n")
	getNymResponse, err := indy.SubmitRequest(poolHandle, getNymRequest)
	if err != nil {
		log.Printf("Failed to send the GET_NYM request: %s", err)
	}
	log.Printf("GET_NYM response: %s\n", getNymResponse)

	// 12.
	// log.Printf("\n12. Comparint Trust Anchor verkey as written by Steward and as retrieved from ledger\n")

	// 13.
	log.Printf("\n13. Closing wallet and pool \n")
	err = indy.CloseWallet(walletHandle)
	if err != nil {
		log.Fatalf("Failed to close the wallet: %s\n", err)
	}
	err = indy.ClosePoolLedger(poolHandle)
	if err != nil {
		log.Fatalf("Failed to close the pool ledger: %s\n", err)
	}

	// 14.
	log.Printf("\n14. Deleting wallet\n")
	err = indy.DeleteWallet(walletName, "")
	if err != nil {
		log.Fatalf("Failed to delete the wallet: %s\n", err)
	}

	// 15.
	log.Printf("\n15. Deleting pool ledger config\n")
	err = indy.DeletePoolLedgerConfig(poolName)
	if err != nil {
		log.Fatalf("Failed to delete pool ledger config: %s\n", err)
	}

	log.Printf("Successfully completed!\n")
}

//
// +async def write_nym_and_query_verkey():
// +        # 12.
// +        print_log('\n12. Comparing Trust Anchor verkey as written by Steward and as retrieved in GET_NYM '
// +                  'response submitted by Client\n')
// +        print_log('Written by Steward: ', trust_anchor_verkey)
// +        verkey_from_ledger = json.loads(get_nym_response['result']['data'])['verkey']
// +        print_log('Queried from ledger: ', verkey_from_ledger)
// +        print_log('Matching: ', verkey_from_ledger == trust_anchor_verkey)
// +
// +        # 13.
// +        print_log('\n13. Closing wallet and pool\n')
// +        await wallet.close_wallet(wallet_handle)
// +        await pool.close_pool_ledger(pool_handle)
// +
// +        # 14.
// +        print_log('\n14. Deleting created wallet\n')
// +        await wallet.delete_wallet(wallet_name, None)
// +
// +        # 15.
// +        print_log('\n15. Deleting pool ledger config\n')
// +        await pool.delete_pool_ledger_config(pool_name)
// +
// +    except IndyError as e:
// +        print('Error occurred: %s' % e)
