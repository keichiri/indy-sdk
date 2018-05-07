use super::wallet::Keys;
use super::encryption::*;
use super::language::{Operator,TargetValue,TagName};


// Performs encryption of WQL query
// WQL query is provided as top-level Operator
// Recursively transforms operators using encrypt_operator function
pub(super) fn encrypt_query(operator: Operator, keys: &Keys) -> Operator {
    operator.transform(&|op: Operator| -> Operator {encrypt_operator(op, keys)})
}


fn encrypt_operator(op: Operator, keys: &Keys) -> Operator {
    match op {
        Operator::Eq(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Eq(encrypted_name, encrypted_value)
        },
        Operator::Neq(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Neq(encrypted_name, encrypted_value)
        },
       Operator::Gt(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Gt(encrypted_name, encrypted_value)
        },
        Operator::Gte(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Gte(encrypted_name, encrypted_value)
        },
        Operator::Lt(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Lt(encrypted_name, encrypted_value)
        },
        Operator::Lte(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Lte(encrypted_name, encrypted_value)
        },
        Operator::Like(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Like(encrypted_name, encrypted_value)
        },
        Operator::Regex(name, value) => {
            let (encrypted_name, encrypted_value) = encrypt_name_value(&name, value, keys);
            Operator::Regex(encrypted_name, encrypted_value)
        },
        Operator::In(name, values) => {
            let name = match name {
                TagName::EncryptedTagName(ref name) => {
                    let encrypted_name = encrypt_as_searchable(&name[..], keys.tag_name_key, keys.tags_hmac_key);
                    TagName::EncryptedTagName(encrypted_name)
                },
                TagName::PlainTagName(ref name) => {
                    let encrypted_name = encrypt_as_searchable(&name[..], keys.tag_name_key, keys.tags_hmac_key);
                    TagName::PlainTagName(encrypted_name)
                }
            };
            let mut encrypted_values: Vec<TargetValue> = Vec::new();

            for value in values {
                encrypted_values.push(encrypt_name_value(&name, value, keys).1);
            }
            Operator::In(name, encrypted_values)
        },
        _ => op
    }
}


// Encrypts a single tag name, tag value pair.
// If the tag name is EncryptedTagName enum variant, encrypts both the tag name and the tag value
// If the tag name is PlainTagName enum variant, encrypts only the tag name
fn encrypt_name_value(name: &TagName, value: TargetValue, keys: &Keys) -> (TagName, TargetValue) {
    match (name, value) {
        (&TagName::EncryptedTagName(ref name), TargetValue::Unencrypted(ref s)) => {
            let encrypted_tag_name = encrypt_as_searchable(&name[..], keys.tag_name_key, keys.tags_hmac_key);
            let encrypted_tag_value = encrypt_as_searchable(s.as_bytes(), keys.tag_value_key, keys.tags_hmac_key);
            (TagName::EncryptedTagName(encrypted_tag_name), TargetValue::Encrypted(encrypted_tag_value))
        },
        (&TagName::PlainTagName(ref name), TargetValue::Unencrypted(ref s)) => {
            let encrypted_tag_name = encrypt_as_searchable(&name[..], keys.tag_name_key, keys.tags_hmac_key);
            (TagName::PlainTagName(encrypted_tag_name), TargetValue::Unencrypted(s.clone()))
        },
        _ => unreachable!()
    }
}