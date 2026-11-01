{ pkgs }:

let
  # MyGo's tray activation requires the signal added in 0.6.0.
  appIndicator = pkgs.libayatana-appindicator.overrideAttrs (
    finalAttrs: previousAttrs: {
      version = "0.6.0";
      src = pkgs.fetchFromGitHub {
        owner = "AyatanaIndicators";
        repo = "libayatana-appindicator";
        rev = finalAttrs.version;
        hash = "sha256-cKrjv7d81XkS4mdBfZXp+F3NUkBRNXGzunHQCQd9SEg=";
      };
      cmakeFlags = previousAttrs.cmakeFlags ++ [ "-DENABLE_GTKDOC=OFF" ];
    }
  );
in
with pkgs;
[
  gtk3
  glib
  pango
  cairo
  harfbuzz
  gdk-pixbuf
  fontconfig
  libepoxy
  libglvnd
  appIndicator
]
