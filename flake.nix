{
  description = "hop — git worktree + tmux session launcher";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        default = pkgs.buildGoModule {
          pname = "hop";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-uwBJAqN4sIepiiJf9lCDumLqfKJEowQO2tOiSWD3Fig=";
        };
      });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.system}.default}/bin/hop";
        };
      });
    };
}
