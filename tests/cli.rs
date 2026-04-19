//! CLI-surface integration tests.
//!
//! These spawn the release binary and assert on stdout/exit codes. They're
//! cheap sanity checks that catch regressions in the argument parser, the
//! asset-install pipeline, and the path-resolution layering.

use std::path::PathBuf;
use std::process::Command;

fn bin() -> PathBuf {
    // Prefer the release build; fall back to debug so `cargo test` works
    // without a separate `cargo build --release` step.
    let release = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("target")
        .join("release")
        .join("ghostty-tui");
    if release.exists() {
        return release;
    }
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("target")
        .join("debug")
        .join("ghostty-tui")
}

fn repo_assets() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("assets")
}

#[test]
fn version_flag_prints_semver() {
    let out = Command::new(bin()).arg("--version").output().unwrap();
    assert!(out.status.success());
    let stdout = String::from_utf8_lossy(&out.stdout);
    assert!(stdout.contains("ghostty-tui v"), "stdout was: {}", stdout);
}

#[test]
fn help_flag_lists_install_assets() {
    let out = Command::new(bin()).arg("--help").output().unwrap();
    assert!(out.status.success());
    let stdout = String::from_utf8_lossy(&out.stdout);
    assert!(stdout.contains("--install-assets"));
    assert!(stdout.contains("--assets-path"));
    assert!(stdout.contains("GHOSTTY_TUI_ASSETS"));
    assert!(stdout.contains("RELOAD GHOSTTY"));
}

#[test]
fn assets_path_resolves_in_repo_layout() {
    let assets = repo_assets();
    let out = Command::new(bin())
        .env("GHOSTTY_TUI_ASSETS", &assets)
        .arg("--assets-path")
        .output()
        .unwrap();
    assert!(out.status.success(), "stderr: {}", String::from_utf8_lossy(&out.stderr));
    let stdout = String::from_utf8_lossy(&out.stdout);
    assert!(stdout.trim().ends_with("assets"), "stdout was: {}", stdout);
}

#[test]
fn install_assets_copies_into_test_home() {
    let tmp = std::env::temp_dir().join("ghostty-tui-install-test");
    let _ = std::fs::remove_dir_all(&tmp);
    std::fs::create_dir_all(&tmp).unwrap();

    let out = Command::new(bin())
        .env("HOME", &tmp)
        .env("GHOSTTY_TUI_ASSETS", repo_assets())
        .arg("--install-assets")
        .output()
        .unwrap();
    assert!(out.status.success(), "stderr: {}", String::from_utf8_lossy(&out.stderr));

    let themes_dir = tmp.join(".config/ghostty/themes");
    let cursors_dir = tmp.join(".config/ghostty/shaders/cursors");
    let screen_dir = tmp.join(".config/ghostty/shaders/screen");
    assert!(themes_dir.is_dir(), "themes dir not created");
    assert!(cursors_dir.is_dir(), "cursors dir not created");
    assert!(screen_dir.is_dir(), "screen dir not created");

    let theme_count = std::fs::read_dir(&themes_dir).unwrap().count();
    let cursor_count = std::fs::read_dir(&cursors_dir).unwrap().count();
    let screen_count = std::fs::read_dir(&screen_dir).unwrap().count();
    assert_eq!(theme_count, 5, "expected 5 themes, got {}", theme_count);
    assert_eq!(cursor_count, 9, "expected 9 cursor shaders, got {}", cursor_count);
    assert_eq!(screen_count, 12, "expected 12 screen shaders, got {}", screen_count);

    // Re-run should be idempotent (skip, not overwrite).
    let out2 = Command::new(bin())
        .env("HOME", &tmp)
        .env("GHOSTTY_TUI_ASSETS", repo_assets())
        .arg("--install-assets")
        .output()
        .unwrap();
    assert!(out2.status.success());
    let stdout2 = String::from_utf8_lossy(&out2.stdout);
    assert!(stdout2.contains("Installed 0 files. Skipped 26"), "stdout2 was: {}", stdout2);

    std::fs::remove_dir_all(&tmp).ok();
}

#[test]
fn unknown_arg_exits_nonzero() {
    let out = Command::new(bin()).arg("--frobnicate").output().unwrap();
    assert!(!out.status.success());
    let stderr = String::from_utf8_lossy(&out.stderr);
    assert!(stderr.contains("Unknown argument"));
}
