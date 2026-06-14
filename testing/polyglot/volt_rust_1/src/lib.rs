#![allow(dead_code)]

#[no_mangle]
pub extern "C" fn rust_check() -> i32 {
    println!("Rust execution context active.");
    42
}
