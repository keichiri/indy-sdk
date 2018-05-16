pub mod default;
pub mod plugged;

use std::collections::HashMap;

use errors::wallet::WalletStorageError;
use services::wallet::language;
use services::wallet::wallet::{TagName, WalletRuntimeConfig};


#[derive(Clone, Debug, PartialEq)]
pub enum TagValue {
    Encrypted(Vec<u8>),
    Plain(String),
}

#[derive(Clone, Debug, PartialEq)]
pub struct StorageValue {
    pub data: Vec<u8>,
    pub key: Vec<u8>
}

#[derive(Clone, Debug, PartialEq)]
pub struct StorageEntity {
    pub name: Vec<u8>,
    pub value: Option<StorageValue>,
    pub type_: Option<Vec<u8>>,
    pub tags: Option<HashMap<Vec<u8>, TagValue>>,
}

impl StorageValue {
    fn new(data: Vec<u8>, key: Vec<u8>) -> Self {
        Self {
            data: data,
            key: key,
        }
    }
}

impl StorageEntity {
    fn new(name: Vec<u8>, value: Option<StorageValue>, type_: Option<Vec<u8>>, tags: Option<HashMap<Vec<u8>, TagValue>>) -> Self {
        Self {
            name: name,
            value: value,
            type_: type_,
            tags: tags,
        }
    }
}


pub trait StorageIterator {
    fn next(&mut self) -> Result<Option<StorageEntity>, WalletStorageError>;
}


pub trait WalletStorage {
    fn get(&self, type_: &Vec<u8>, name: &Vec<u8>, options: &str) -> Result<StorageEntity, WalletStorageError>;
    fn add(&self, type_: &Vec<u8>, name: &Vec<u8>, value: &Vec<u8>, value_key: &Vec<u8>, tags: &HashMap<Vec<u8>, TagValue>) -> Result<(), WalletStorageError>;
    fn add_tags(&mut self, type_: &Vec<u8>, name: &Vec<u8>, tags: &HashMap<Vec<u8>, TagValue>) -> Result<(), WalletStorageError>;
    fn update_tags(&mut self, type_: &Vec<u8>, name: &Vec<u8>, tags: &HashMap<Vec<u8>, TagValue>) -> Result<(), WalletStorageError>;
    fn delete_tags(&mut self, type_: &Vec<u8>, name: &Vec<u8>, tag_names: &[TagName]) -> Result<(), WalletStorageError>;
    fn update(&self, type_: &Vec<u8>, name: &Vec<u8>, value: &Vec<u8>, value_key: &Vec<u8>) -> Result<(), WalletStorageError>;
    fn delete(&self, type_: &Vec<u8>, name: &Vec<u8>) -> Result<(), WalletStorageError>;
    fn get_all<'a>(&'a self) -> Result<Box<StorageIterator + 'a>, WalletStorageError>;
    fn search<'a>(&'a self, type_: &Vec<u8>, query: &language::Operator, options: &str) -> Result<Box<StorageIterator + 'a>, WalletStorageError>;
    fn clear(&self) -> Result<(), WalletStorageError>;
    fn close(&mut self) -> Result<(), WalletStorageError>;
}


pub trait WalletStorageType {
    fn create_storage(&self, name: &str, config: Option<&str>, credentials: &str, keys: &Vec<u8>) -> Result<(), WalletStorageError>;
    fn open_storage(&self, name: &str, config: Option<&str>, credentials: &str) -> Result<(Box<WalletStorage>, Vec<u8>), WalletStorageError>;
    fn delete_storage(&self, name: &str, config: Option<&str>, credentials: &str) -> Result<(), WalletStorageError>;
}