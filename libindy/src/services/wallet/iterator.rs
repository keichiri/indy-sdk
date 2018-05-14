use serde_json;

use errors::wallet::WalletError;

use super::WalletRecord;
use super::wallet::Keys;
use super::storage::StorageIterator;
use super::encryption::{decrypt_tags};
use utils::crypto::chacha20poly1305_ietf::ChaCha20Poly1305IETF;


pub(super) struct WalletIterator<'a> {
    storage_iterator: Box<StorageIterator + 'a>,
    keys: &'a Keys
}


impl<'a> WalletIterator<'a> {
    pub fn new(storage_iter: Box<StorageIterator + 'a>, keys: &'a Keys) -> Self {
        WalletIterator {
            storage_iterator: storage_iter,
            keys: keys
        }
    }

    pub fn next(&mut self) -> Result<Option<WalletRecord>, WalletError> {
        let next_storage_entity = self.storage_iterator.next()?;
        if let Some(next_storage_entity) = next_storage_entity {
            let decrypted_name = ChaCha20Poly1305IETF::decrypt(&next_storage_entity.name, &self.keys.name_key)?;
            let name = String::from_utf8(decrypted_name)?;

            let value = match next_storage_entity.value {
                None => None,
                Some(storage_value) => {
                    let value_key = ChaCha20Poly1305IETF::decrypt(&storage_value.key, &self.keys.value_key)?;
                    if value_key.len() != ChaCha20Poly1305IETF::key_len() {
                        return Err(WalletError::EncryptionError("Value key is not right size".to_string()));
                    }
                    Some(String::from_utf8(ChaCha20Poly1305IETF::decrypt(&storage_value.data, &value_key)?)?)
                }
            };

            let tags = match decrypt_tags(&next_storage_entity.tags, &self.keys.tag_name_key, &self.keys.tag_value_key)?;

            Ok(Some(WalletRecord::new(name, None, value, tags)))
        } else { Ok(None) }
    }
}