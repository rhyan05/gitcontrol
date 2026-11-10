// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        // plugin-shell é quem executa o sidecar gitcontrol --json (Bloco 9b).
        .plugin(tauri_plugin_shell::init())
        // plugin-dialog permite escolher a pasta do repositório.
        .plugin(tauri_plugin_dialog::init())
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
