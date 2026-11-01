{ pkgs }:

let
  inherit (pkgs) lib;
  app = builtins.fromJSON (builtins.readFile ../mygo.json);
  desktopLibraries = import ./desktop-libraries.nix { inherit pkgs; };
  arch = pkgs.stdenv.hostPlatform.go.GOARCH;
in
pkgs.buildGo127Module {
  pname = "biternet";
  inherit (app) version;

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../go.sum
      ../main.go
      ../mygo.json
      ../client
      (lib.fileset.fileFilter (file: !lib.hasSuffix ".ttf.gz" file.name) ../internal)
      ../resources
      ../scripts
      ../LICENSE
    ];
  };

  vendorHash = "sha256-FXQNsT04q7iAg4Aa5BTXEZmCOtwgEkkpTPxikONIn9k=";
  proxyVendor = true;
  env.CGO_ENABLED = 0;

  nativeBuildInputs = [
    pkgs.makeWrapper
    (pkgs.python3.withPackages (ps: [ ps.fonttools ]))
  ];
  nativeCheckInputs = desktopLibraries;

  buildPhase = ''
    runHook preBuild
    go generate ./internal/desktop
    go tool mygo build -platform linux/${arch}
    runHook postBuild
  '';

  preCheck = ''
    export LD_LIBRARY_PATH="${lib.makeLibraryPath desktopLibraries}"
    export XDG_CACHE_HOME="$TMPDIR/cache"
    export FONTCONFIG_FILE="${pkgs.fontconfig.out}/etc/fonts/fonts.conf"
    mkdir -p "$XDG_CACHE_HOME"
  '';
  checkPhase = ''
    runHook preCheck
    go test ./...
    (cd client && go test -mod=mod ./...)
    runHook postCheck
  '';

  installPhase = ''
    runHook preInstall
    install -Dm755 build/linux-${arch}/biternet "$out/bin/biternet"
    install -Dm644 build/linux-${arch}/biternet.desktop "$out/share/applications/biternet.desktop"
    echo 'StartupWMClass=${app.identifier}' >> "$out/share/applications/biternet.desktop"
    install -Dm644 build/linux-${arch}/biternet-256x256.png "$out/share/icons/hicolor/256x256/apps/biternet.png"
    install -Dm644 build/linux-${arch}/biternet-512x512.png "$out/share/icons/hicolor/512x512/apps/biternet.png"
    install -Dm644 LICENSE "$out/share/licenses/biternet/LICENSE"
    install -m644 internal/desktop/fonts/*-OFL.txt "$out/share/licenses/biternet/"
    wrapProgram "$out/bin/biternet" \
      --prefix PATH : ${lib.makeBinPath [ pkgs.xdg-utils ]} \
      --prefix LD_LIBRARY_PATH : "/run/opengl-driver/lib:${lib.makeLibraryPath desktopLibraries}"
    runHook postInstall
  '';

  meta = {
    description = "BIT campus network client built with Go and MyGo";
    homepage = "https://github.com/bitoolkit/biternet-go";
    license = lib.licenses.unlicense;
    mainProgram = "biternet";
    platforms = lib.platforms.linux;
  };
}
