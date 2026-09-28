#!/usr/bin/env bash
# Installs the app for the current user on Linux: the binary on PATH as
# investec-tui, its icon, and a desktop entry so app launchers can find it.
#
# The binary is linked, not copied, so ./run keeps the installed command up to
# date. Re-run this after moving the checkout.
set -euo pipefail

cd "$(dirname "$0")/.."

bin_dir="${XDG_BIN_HOME:-$HOME/.local/bin}"
data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
icon_dir="$data_dir/icons/hicolor/256x256/apps"
desktop_file="$data_dir/applications/investec-tui.desktop"

go build -o investec.openbanking.tui .

mkdir -p "$bin_dir" "$icon_dir" "$(dirname "$desktop_file")"
ln -sf "$PWD/investec.openbanking.tui" "$bin_dir/investec-tui"
cp packaging/linux/investec-tui.png "$icon_dir/investec-tui.png"
gtk-update-icon-cache "$data_dir/icons/hicolor" &>/dev/null || true

# On Omarchy, launch the way its own TUIs are: in the themed terminal, under
# the app-id org.omarchy.investec-tui, focusing the window if it is already
# open. Elsewhere, let the desktop pick a terminal.
if command -v omarchy-launch-or-focus-tui >/dev/null; then
  exec_line="omarchy-launch-or-focus-tui investec-tui"
  terminal=false
else
  exec_line="investec-tui"
  terminal=true
fi

cat >"$desktop_file" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=Investec
GenericName=Banking
Comment=Accounts, balances, transactions and statements from Investec Open Banking
Exec=$exec_line
Terminal=$terminal
Icon=investec-tui
Categories=Office;Finance;
Keywords=bank;investec;balance;transactions;
StartupNotify=true
EOF

echo "Installed $bin_dir/investec-tui and $desktop_file"

# On Omarchy, give the window the same treatment as btop and Omarchy's own
# terminal windows: floating, centred, at the standard floating size. Hyprland
# has no drop-in folder for user rules, so the rule is appended to
# hyprland.lua once and left alone after that.
hypr_config="${XDG_CONFIG_HOME:-$HOME/.config}/hypr/hyprland.lua"
window_rule='o.window("org.omarchy.investec-tui", { tag = "+floating-window" })'

if command -v omarchy-launch-or-focus-tui >/dev/null && [[ -f $hypr_config ]]; then
  if ! grep -qF 'org.omarchy.investec-tui' "$hypr_config"; then
    cat >>"$hypr_config" <<EOF

-- Investec TUI floats, centred, at Omarchy's standard floating size.
$window_rule
EOF
    echo "Added the Investec window rule to $hypr_config"
  fi

  if command -v hyprctl >/dev/null && hyprctl reload >/dev/null 2>&1; then
    errors=$(hyprctl configerrors 2>/dev/null || true)
    if [[ -n $errors && $errors != *"no errors"* ]]; then
      echo "Hyprland reported config errors:" >&2
      echo "$errors" >&2
    fi
  fi
fi
