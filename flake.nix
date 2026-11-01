{
  description = "biternet-go: BIT campus network client built with Go and MyGo";

  nixConfig = {
    extra-substituters = [ "https://so1ve.cachix.org" ];
    extra-trusted-public-keys = [
      "so1ve.cachix.org-1:51jcW4FkJhiLcqPsiUx3nglRP469les8F9zjFxio1nw="
    ];
  };

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      mkPackage = pkgs: import ./nix/package.nix { inherit pkgs; };
    in
    {
      packages = forAllSystems (
        system:
        let
          biternet = mkPackage nixpkgs.legacyPackages.${system};
        in
        {
          inherit biternet;
          default = biternet;
        }
      );

      apps = forAllSystems (
        system:
        let
          app = {
            type = "app";
            program = nixpkgs.lib.getExe self.packages.${system}.default;
            meta.description = self.packages.${system}.default.meta.description;
          };
        in
        {
          biternet = app;
          default = app;
        }
      );

      checks = forAllSystems (system: {
        package = self.packages.${system}.default;
      });

      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt-tree);
      overlays.default = final: _prev: { biternet = mkPackage final; };
      homeManagerModules.default = import ./nix/home-manager-module.nix self;
      nixosModules.default = import ./nix/nixos-module.nix self;
    };
}
