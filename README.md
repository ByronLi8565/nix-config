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
   chezmoi --version
   ```

9. Apply future nixfile changes with:

   ```sh
   rebuild
   ```

10. Set up dotfiles with `chezmoi`:

    ```sh
    chezmoi init --apply https://github.com/ByronLi8565/dotfiles.git
    ```

11. If dotfiles are needed before the first rebuild, run `chezmoi` directly from nixpkgs:

    ```sh
    nix run nixpkgs#chezmoi -- init --apply https://github.com/ByronLi8565/dotfiles.git
    ```

12. Reboot.
