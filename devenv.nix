{
  inputs,
  lib,
  pkgs,
  ...
}:

let
  desktopLibraries = import ./nix/desktop-libraries.nix { inherit pkgs; };
in
{
  imports = [
    (inputs.ray-devenv + "/profiles/config.nix")
    (inputs.ray-devenv + "/profiles/go.nix")
    (inputs.ray-devenv + "/profiles/nix.nix")
  ]
  ++ lib.optional (builtins.pathExists ./devenv.local.nix) ./devenv.local.nix;

  packages = lib.optionals pkgs.stdenv.isLinux (
    desktopLibraries
    ++ [
      pkgs.stdenv.cc
      pkgs.xdg-utils
    ]
  );

  # MyGo loads native libraries at runtime, without linking through CGO.
  env.LD_LIBRARY_PATH = lib.mkIf pkgs.stdenv.isLinux (
    "/run/opengl-driver/lib:${lib.makeLibraryPath desktopLibraries}"
  );
}
