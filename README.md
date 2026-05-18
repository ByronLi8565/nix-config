# macOS Setup

1. Install macOS updates from System Settings.

2. Install Xcode command line tools:

   ```sh
   xcode-select --install
   ```

3. Install Determinate Nix:

   ```sh
   curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install --determinate
   ```

4. Open a new terminal and verify Nix:

   ```sh
   nix --version
   ```

5. Clone these nixfiles:

   ```sh
   git clone <nix-config-repo-url> ~/nix-config
   cd ~/nix-config
   ```

6. If this is a different Mac name or username, add a host under `hosts/` and update `name`, `user`, and `system` in that host file. For the current config, use `spheal-mbp`.

7. Apply the nix-darwin configuration:

   ```sh
   sudo nix run nix-darwin/master#darwin-rebuild -- switch --flake ~/nix-config#spheal-mbp
   ```

8. Open a new terminal and confirm the installed tools are available:

   ```sh
   darwin-rebuild --version
   home-manager --version
   nun --help
   ```

9. Apply future nixfile changes with:

   ```sh
   rebuild
   ```

10. Link dotfiles from this repo:

    ```sh
    nun link
    ```

    Fish config is managed by Home Manager from `dotfiles/fish/` and is applied
    by `nun rebuild`.

11. If dotfiles are needed before the first rebuild, link them from the checkout directly:

    ```sh
    ln -s "$PWD/dotfiles/aerospace.toml" ~/.aerospace.toml
    ln -s "$PWD/dotfiles/skhdrc" ~/.skhdrc
    ln -s "$PWD/dotfiles/yabairc" ~/.yabairc
    mkdir -p ~/.config/ghostty ~/.config/nvim
    ln -s "$PWD/dotfiles/ghostty/config" ~/.config/ghostty/config
    ln -s "$PWD/dotfiles/nvim/init.lua" ~/.config/nvim/init.lua
    ```

12. Reboot.
