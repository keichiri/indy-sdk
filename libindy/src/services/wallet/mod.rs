extern crate libc;
extern crate indy_crypto;
extern crate rusqlite;

mod storage;
mod encryption;
mod wallet;
mod iterator;
mod query_encryption;
mod language;

use std::cell::RefCell;
use std::collections::HashMap;
use std::fs;
use std::fs::{File, DirBuilder};
use std::io::{Read, Write};
use std::path::PathBuf;

use base64;
use serde_json;

use api::ErrorCode;
use errors::wallet::WalletError;
use utils::environment::EnvironmentUtils;
use utils::sequence::SequenceUtils;

use self::indy_crypto::utils::json::{JsonDecodable, JsonEncodable};
use self::encryption::decrypt;
use self::storage::default::SQLiteStorageType;
use self::storage::StorageType;
use self::wallet::{Wallet, WalletRecord, Keys, Tags};
use self::libc::c_char;


#[derive(Serialize, Deserialize)]
struct WalletDescriptor {
    pool_name: String,
    xtype: String,
    name: String
}

impl WalletDescriptor {
    pub fn new(pool_name: &str, xtype: &str, name: &str) -> WalletDescriptor {
        WalletDescriptor {
            pool_name: pool_name.to_string(),
            xtype: xtype.to_string(),
            name: name.to_string()
        }
    }
}

impl JsonEncodable for WalletDescriptor {}

impl<'a> JsonDecodable<'a> for WalletDescriptor {}


#[derive(Serialize, Deserialize, Debug, Eq, PartialEq)]
pub struct WalletMetadata {
    name: String,
    #[serde(rename = "type")]
    type_: String,
    associated_pool_name: String,
}

impl From<WalletDescriptor> for WalletMetadata {
    fn from(internal: WalletDescriptor) -> Self {
        WalletMetadata {
            name: internal.name,
            type_: internal.xtype,
            associated_pool_name: internal.pool_name,
        }
    }
}

impl JsonEncodable for WalletMetadata {}

impl<'a> JsonDecodable<'a> for WalletMetadata {}


#[derive(Debug)]
pub struct WalletCredentials {
    master_key: [u8; 32],
    storage_credentials: String,
}


impl WalletCredentials {
    fn from_json(json: &str) -> Result<WalletCredentials, WalletError> {
        if let serde_json::Value::Object(m) = try!(serde_json::from_str(json)) {
            let master_key = if let Some(&serde_json::Value::String(ref master_key_encoded)) = m.get("master_key") {
                let decoded_vector = try!(base64::decode(&master_key_encoded));
                let mut master_key: [u8; 32] = [0; 32];
                master_key.clone_from_slice(&decoded_vector[0..32]);
                master_key
            } else {
                return Err(WalletError::InputError(String::from("Credentials missing 'master_key' field")));
            };

            let storage_credentials = if let Some(&serde_json::Value::Object(ref storage_credentials)) = m.get("storage_credentials") {
                serde_json::to_string(&storage_credentials).unwrap()
            } else {
                return Err(WalletError::InputError(String::from("Credentials missing 'storage_credentials' field")));
            };

            Ok(WalletCredentials {
                master_key: master_key,
                storage_credentials: storage_credentials
            })
        } else {
            return Err(WalletError::InputError(String::from("Credentials must be JSON object")));
        }
    }
}



pub struct WalletService {
    storage_types: RefCell<HashMap<String, Box<StorageType>>>,
    wallets: RefCell<HashMap<i32, Box<Wallet>>>
}

impl WalletService {
    pub fn new() -> WalletService {
        let mut types: HashMap<String, Box<StorageType>> = HashMap::new();
        types.insert("default".to_string(), Box::new(SQLiteStorageType::new()));

        WalletService {
            storage_types: RefCell::new(types),
            wallets: RefCell::new(HashMap::new())
        }
    }

    pub fn register_type(&self,
                         xtype: &str,
                         create: extern fn(name: *const c_char,
                                           config: *const c_char,
                                           credentials: *const c_char) -> ErrorCode,
                         open: extern fn(name: *const c_char,
                                         config: *const c_char,
                                         runtime_config: *const c_char,
                                         credentials: *const c_char,
                                         handle: *mut i32) -> ErrorCode,
                         set: extern fn(handle: i32,
                                        key: *const c_char,
                                        value: *const c_char) -> ErrorCode,
                         get: extern fn(handle: i32,
                                        key: *const c_char,
                                        value_ptr: *mut *const c_char) -> ErrorCode,
                         get_not_expired: extern fn(handle: i32,
                                                    key: *const c_char,
                                                    value_ptr: *mut *const c_char) -> ErrorCode,
                         list: extern fn(handle: i32,
                                         key_prefix: *const c_char,
                                         values_json_ptr: *mut *const c_char) -> ErrorCode,
                         close: extern fn(handle: i32) -> ErrorCode,
                         delete: extern fn(name: *const c_char,
                                           config: *const c_char,
                                           credentials: *const c_char) -> ErrorCode,
                         free: extern fn(wallet_handle: i32,
                                         value: *const c_char) -> ErrorCode) -> Result<(), WalletError> {
        let mut wallet_types = self.storage_types.borrow_mut();

        if wallet_types.contains_key(xtype) {
            return Err(WalletError::TypeAlreadyRegistered(xtype.to_string()));
        }

        // TODO - uncomment once plugged wallet is implemented
//        wallet_types.insert(xtype.to_string(),
//                            Box::new(
//                                PluggedWalletType::new(create, open, set, get,
//                                                       get_not_expired, list, close, delete, free)));
        Ok(())
    }

    pub fn create_wallet(&self,
                         pool_name: &str,
                         name: &str,
                         storage_type_name: Option<&str>,
                         storage_config: Option<&str>,
                        credentials: &str) -> Result<(), WalletError> {
        let storage_type_name = storage_type_name.unwrap_or("default");

        let storage_types = self.storage_types.borrow();
        if !storage_types.contains_key(storage_type_name) {
            return Err(WalletError::UnknownType(storage_type_name.to_string()));
        }

        let wallet_path = _wallet_path(name);
        if wallet_path.exists() {
            return Err(WalletError::AlreadyExists(name.to_string()));
        }
        DirBuilder::new()
            .recursive(true)
            .create(wallet_path)?;

        let storage_type = storage_types.get(storage_type_name).unwrap();
        let credentials = WalletCredentials::from_json(credentials)?;
        storage_type.create(name, storage_config, &credentials.storage_credentials, &Keys::gen_keys(credentials.master_key))?;

        let mut descriptor_file = File::create(_wallet_descriptor_path(name))?;
        descriptor_file
            .write_all({
                WalletDescriptor::new(pool_name, storage_type_name, name)
                    .to_json()?
                    .as_bytes()
            })?;
        descriptor_file.sync_all()?;

        if storage_config.is_some() {
            let mut config_file = File::create(_wallet_config_path(name))?;
            config_file.write_all(storage_config.unwrap().as_bytes())?;
            config_file.sync_all()?;
        }

        Ok(())
    }

    pub fn delete_wallet(&self, name: &str, credentials: &str) -> Result<(), WalletError> {
        let mut descriptor_json = String::new();
        let descriptor: WalletDescriptor = WalletDescriptor::from_json({
            let mut file = File::open(_wallet_descriptor_path(name))?; // FIXME: Better error!
            file.read_to_string(&mut descriptor_json)?;
            descriptor_json.as_str()
        })?;

        let wallet_types = self.storage_types.borrow();
        if !wallet_types.contains_key(descriptor.xtype.as_str()) {
            return Err(WalletError::UnknownType(descriptor.xtype));
        }

        let wallet_type = wallet_types.get(descriptor.xtype.as_str()).unwrap();

        let config = {
            let config_path = _wallet_config_path(name);

            if config_path.exists() {
                let mut config_json = String::new();
                let mut file = File::open(config_path)?;
                file.read_to_string(&mut config_json)?;
                Some(config_json)
            } else {
                None
            }
        };

        wallet_type.delete(name,
                           config.as_ref().map(String::as_str),
                           credentials)?;

        fs::remove_dir_all(_wallet_path(name))?;
        Ok(())
    }

    pub fn open_wallet(&self, name: &str, credentials: &str) -> Result<i32, WalletError> {
        let mut descriptor_json = String::new();
        let descriptor: WalletDescriptor = WalletDescriptor::from_json({
            let mut file = File::open(_wallet_descriptor_path(name))?; // FIXME: Better error!
            file.read_to_string(&mut descriptor_json)?;
            descriptor_json.as_str()
        })?;

        let storage_types = self.storage_types.borrow();
        if !storage_types.contains_key(descriptor.xtype.as_str()) {
            return Err(WalletError::UnknownType(descriptor.xtype));
        }
        let storage_type = storage_types.get(descriptor.xtype.as_str()).unwrap();

        let mut wallets = self.wallets.borrow_mut();
        if wallets.values().any(|ref wallet| wallet.get_name() == name) {
            return Err(WalletError::AlreadyOpened(name.to_string()));
        }

        let config = {
            let config_path = _wallet_config_path(name);

            if config_path.exists() {
                let mut config_json = String::new();
                let mut file = File::open(config_path)?;
                file.read_to_string(&mut config_json)?;
                Some(config_json)
            } else {
                None
            }
        };

        let credentials = WalletCredentials::from_json(credentials)?;

        let (storage, enc_keys) = storage_type.open(name,
                                                    config.as_ref().map(String::as_str),
                                                    &credentials.storage_credentials)?;
        let key_vector = decrypt(&enc_keys, credentials.master_key)?;
        let keys = Keys::new(key_vector);
        let wallet = Box::new(Wallet::new(name, &descriptor.pool_name, storage, keys));

        let wallet_handle = SequenceUtils::get_next_id();
        wallets.insert(wallet_handle, wallet);
        Ok(wallet_handle)
    }

    pub fn list_wallets(&self) -> Result<Vec<WalletMetadata>, WalletError> {
        let mut descriptors = Vec::new();
        let wallet_home_path = EnvironmentUtils::wallet_home_path();

        for entry in fs::read_dir(wallet_home_path)? {
            let dir_entry = if let Ok(dir_entry) = entry { dir_entry } else { continue };
            if let Some(wallet_name) = dir_entry.path().file_name().and_then(|os_str| os_str.to_str()) {
                let mut descriptor_json = String::new();
                File::open(_wallet_descriptor_path(wallet_name)).ok()
                    .and_then(|mut f| f.read_to_string(&mut descriptor_json).ok())
                    .and_then(|_| WalletDescriptor::from_json(descriptor_json.as_str()).ok())
                    .map(|descriptor| descriptors.push(descriptor.into()));
            }
        }

        Ok(descriptors)
    }

    pub fn close_wallet(&self, handle: i32) -> Result<(), WalletError> {
        match self.wallets.borrow_mut().remove(&handle) {
            Some(mut wallet) => wallet.close(),
            None => Err(WalletError::InvalidHandle(handle.to_string()))
        }
    }

    pub fn add_record(&self, handle: i32, type_: &str, name: &str, value: &str, tags: &str) -> Result<(), WalletError> {
        match self.wallets.borrow().get(&handle) {
            Some(wallet) => {
                let tags: Tags = serde_json::from_str(tags)?;
                wallet.add(type_, name, value, &tags)
            },
            None => Err(WalletError::InvalidHandle(handle.to_string()))
        }
    }

    pub fn get_record(&self, handle: i32, type_: &str, name: &str, options: &str) -> Result<WalletRecord, WalletError> {
        match self.wallets.borrow().get(&handle) {
            Some(wallet) => wallet.get(type_, name, options),
            None => Err(WalletError::InvalidHandle(handle.to_string()))
        }
    }

//    pub fn list(&self, handle: i32, key_prefix: &str) -> Result<Vec<(String, String)>, WalletError> {
//        match self.wallets.borrow().get(&handle) {
//            Some(wallet) => wallet.list(key_prefix),
//            None => Err(WalletError::InvalidHandle(handle.to_string()))
//        }
//    }


    pub fn get_pool_name(&self, handle: i32) -> Result<String, WalletError> {
        match self.wallets.borrow().get(&handle) {
            Some(wallet) => Ok(wallet.get_pool_name()),
            None => Err(WalletError::InvalidHandle(handle.to_string()))
        }
    }
}

fn _wallet_path(name: &str) -> PathBuf {
    EnvironmentUtils::wallet_path(name)
}

fn _wallet_descriptor_path(name: &str) -> PathBuf {
    _wallet_path(name).join("wallet.json")
}

fn _wallet_config_path(name: &str) -> PathBuf {
    _wallet_path(name).join("config.json")
}

//
//#[cfg(test)]
//mod tests {
//    use super::*;
//    use errors::wallet::WalletError;
////    use utils::inmem_wallet::InmemWallet;
//    use utils::test::TestUtils;
//
//    use std::time::Duration;
//    use std::thread;
//
//    #[test]
//    fn wallet_service_new_works() {
//        WalletService::new();
//    }
//
//    #[test]
//    fn wallet_service_register_type_works() {
//        TestUtils::cleanup_indy_home();
//        InmemWallet::cleanup();
//
//        let wallet_service = WalletService::new();
//
//        wallet_service
//            .register_type(
//                "inmem",
//                InmemWallet::create,
//                InmemWallet::open,
//                InmemWallet::set,
//                InmemWallet::get,
//                InmemWallet::get_not_expired,
//                InmemWallet::list,
//                InmemWallet::close,
//                InmemWallet::delete,
//                InmemWallet::free
//            )
//            .unwrap();
//
//        TestUtils::cleanup_indy_home();
//        InmemWallet::cleanup();
//    }
//
//    #[test]
//    fn wallet_service_create_wallet_works() {
//        TestUtils::cleanup_indy_home();
//
//        let wallet_service = WalletService::new();
//        wallet_service.create("pool1", Some("default"), "wallet1", None, None).unwrap();
//
//        TestUtils::cleanup_indy_home();
//    }
////
////    #[test]
////    fn wallet_service_create_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_create_works_for_none_type() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_create_works_for_unknown_type() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        let res = wallet_service.create("pool1", Some("unknown"), "wallet1", None, None);
////        assert_match!(Err(WalletError::UnknownType(_)), res);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_create_works_for_twice() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////
////        let res = wallet_service.create("pool1", None, "wallet1", None, None);
////        assert_match!(Err(WalletError::AlreadyExists(_)), res);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_delete_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        wallet_service.delete("wallet1", None).unwrap();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_delete_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        wallet_service.delete("wallet1", None).unwrap();
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_open_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        wallet_service.open("wallet1", None, None).unwrap();
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_open_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        wallet_service.open("wallet1", None, None).unwrap();
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_list_wallets_works() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////        let w1_meta = WalletMetadata {
////            name: "w1".to_string(),
////            associated_pool_name: "p1".to_string(),
////            type_: "default".to_string(),
////        };
////        let w2_meta = WalletMetadata {
////            name: "w2".to_string(),
////            associated_pool_name: "p2".to_string(),
////            type_: "inmem".to_string(),
////        };
////        let w3_meta = WalletMetadata {
////            name: "w3".to_string(),
////            associated_pool_name: "p1".to_string(),
////            type_: "default".to_string(),
////        };
////        wallet_service.create(&w1_meta.associated_pool_name,
////                              Some(&w1_meta.type_),
////                              &w1_meta.name,
////                              None, None).unwrap();
////        wallet_service.create(&w2_meta.associated_pool_name,
////                              Some(&w2_meta.type_),
////                              &w2_meta.name,
////                              None, None).unwrap();
////        wallet_service.create(&w3_meta.associated_pool_name,
////                              None,
////                              &w3_meta.name,
////                              None, None).unwrap();
////
////        let wallets = wallet_service.list_wallets().unwrap();
////
////        assert!(wallets.contains(&w1_meta));
////        assert!(wallets.contains(&w2_meta));
////        assert!(wallets.contains(&w3_meta));
////
////        InmemWallet::cleanup();
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_close_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////        wallet_service.close(wallet_handle).unwrap();
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_close_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////        wallet_service.close(wallet_handle).unwrap();
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_set_get_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_set_get_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_set_get_works_for_reopen() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////        wallet_service.close(wallet_handle).unwrap();
////
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_get_works_for_unknown() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        let res = wallet_service.get(wallet_handle, "key1");
////        assert_match!(Err(WalletError::NotFound(_)), res);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_get_works_for_plugged_and_unknown() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        let res = wallet_service.get(wallet_handle, "key1");
////        assert_match!(Err(WalletError::PluggedWallerError(ErrorCode::WalletNotFoundError)), res);
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_set_get_works_for_update() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        wallet_service.set(wallet_handle, "key1", "value2").unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value2", value);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_set_get_works_for_plugged_and_update() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        wallet_service.set(wallet_handle, "key1", "value2").unwrap();
////        let value = wallet_service.get(wallet_handle, "key1").unwrap();
////        assert_eq!("value2", value);
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_set_get_not_expired_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", Some("{\"freshness_time\": 10}"), None).unwrap();
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////
////
////        let value = wallet_service.get_not_expired(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_set_get_not_expired_works_for_expired() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", Some("{\"freshness_time\": 1}"), None).unwrap();
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////
////        // Wait until value expires
////        thread::sleep(Duration::new(2, 0));
////
////        let res = wallet_service.get_not_expired(wallet_handle, "key1");
////        assert_match!(Err(WalletError::NotFound(_)), res);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_set_get_not_expired_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", Some("{\"freshness_time\": 10}"), None).unwrap();
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////
////        let value = wallet_service.get_not_expired(wallet_handle, "key1").unwrap();
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_set_get_not_expired_works_for_plugged_and_expired() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", Some("{\"freshness_time\": 1}"), None).unwrap();
////        wallet_service.set(wallet_handle, "key1", "value1").unwrap();
////
////        // Wait until value expires
////        thread::sleep(Duration::new(2, 0));
////
////        let res = wallet_service.get_not_expired(wallet_handle, "key1");
////        assert_match!(Err(WalletError::PluggedWallerError(ErrorCode::WalletNotFoundError)), res);
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_list_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        wallet_service.create("pool1", None, "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", Some("{\"freshness_time\": 1}"), None).unwrap();
////
////        wallet_service.set(wallet_handle, "key1::subkey1", "value1").unwrap();
////        wallet_service.set(wallet_handle, "key1::subkey2", "value2").unwrap();
////
////        let mut key_values = wallet_service.list(wallet_handle, "key1::").unwrap();
////        key_values.sort();
////        assert_eq!(2, key_values.len());
////
////        let (key, value) = key_values.pop().unwrap();
////        assert_eq!("key1::subkey2", key);
////        assert_eq!("value2", value);
////
////        let (key, value) = key_values.pop().unwrap();
////        assert_eq!("key1::subkey1", key);
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_list_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", Some("{\"freshness_time\": 1}"), None).unwrap();
////
////        wallet_service.set(wallet_handle, "key1::subkey1", "value1").unwrap();
////        wallet_service.set(wallet_handle, "key1::subkey2", "value2").unwrap();
////
////        let mut key_values = wallet_service.list(wallet_handle, "key1::").unwrap();
////        key_values.sort();
////        assert_eq!(2, key_values.len());
////
////        let (key, value) = key_values.pop().unwrap();
////        assert_eq!("key1::subkey2", key);
////        assert_eq!("value2", value);
////
////        let (key, value) = key_values.pop().unwrap();
////        assert_eq!("key1::subkey1", key);
////        assert_eq!("value1", value);
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_get_pool_name_works() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        let wallet_name = "wallet1";
////        let pool_name = "pool1";
////        wallet_service.create(pool_name, None, wallet_name, None, None).unwrap();
////        let wallet_handle = wallet_service.open(wallet_name, None, None).unwrap();
////
////        assert_eq!(wallet_service.get_pool_name(wallet_handle).unwrap(), pool_name);
////
////        TestUtils::cleanup_indy_home();
////    }
////
////    #[test]
////    fn wallet_service_get_pool_name_works_for_plugged() {
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////
////        let wallet_service = WalletService::new();
////
////        wallet_service
////            .register_type(
////                "inmem",
////                InmemWallet::create,
////                InmemWallet::open,
////                InmemWallet::set,
////                InmemWallet::get,
////                InmemWallet::get_not_expired,
////                InmemWallet::list,
////                InmemWallet::close,
////                InmemWallet::delete,
////                InmemWallet::free
////            )
////            .unwrap();
////
////        wallet_service.create("pool1", Some("inmem"), "wallet1", None, None).unwrap();
////        let wallet_handle = wallet_service.open("wallet1", None, None).unwrap();
////
////        assert_eq!(wallet_service.get_pool_name(wallet_handle).unwrap(), "pool1");
////
////        TestUtils::cleanup_indy_home();
////        InmemWallet::cleanup();
////    }
////
////    #[test]
////    fn wallet_service_get_pool_name_works_for_incorrect_wallet_handle() {
////        TestUtils::cleanup_indy_home();
////
////        let wallet_service = WalletService::new();
////        let wallet_name = "wallet1";
////        let pool_name = "pool1";
////        wallet_service.create(pool_name, None, wallet_name, None, None).unwrap();
////
////        let get_pool_name_res = wallet_service.get_pool_name(1);
////        assert_match!(Err(WalletError::InvalidHandle(_)), get_pool_name_res);
////
////        TestUtils::cleanup_indy_home();
////    }
//}