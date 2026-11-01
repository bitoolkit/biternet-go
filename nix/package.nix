{ pkgs }:

let
  inherit (pkgs) lib;
  app = builtins.fromJSON (builtins.readFile ../mygo.json);
  desktopLibraries = import ./desktop-libraries.nix { inherit pkgs; };
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
      ../client
      ../internal
      ../resources
      ../LICENSE
    ];
  };

  vendorHash = "sha256-BP9t95VyPHHHdyO91kb+iNS5jB9D0NsqYbxFRY8ilVg=";
  proxyVendor = true;
  subPackages = [ "." ];
  env.CGO_ENABLED = 0;
  tags = [ "mygo_noinspector" ];
  ldflags = [
    "-s"
    "-w"
    "-X github.com/egoist/mygo.production=1"
    "-X github.com/egoist/mygo.packageName=${app.name}"
    "-X github.com/egoist/mygo.packageVersion=${app.version}"
    "-X github.com/egoist/mygo.packageIdentifier=${app.identifier}"
  ];

  nativeBuildInputs = [
    pkgs.makeWrapper
    pkgs.copyDesktopItems
  ];
  nativeCheckInputs = desktopLibraries;

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

  desktopItems = [
    (pkgs.makeDesktopItem {
      name = "biternet";
      desktopName = app.name;
      exec = "biternet";
      icon = "biternet";
      comment = app.linux.comment;
      categories = app.linux.categories;
      startupWMClass = app.identifier;
    })
  ];

  postInstall = ''
    mv "$out/bin/biternet-go" "$out/bin/biternet"
    install -Dm644 resources/icon.png "$out/share/icons/hicolor/256x256/apps/biternet.png"
    install -Dm644 LICENSE "$out/share/licenses/biternet/LICENSE"
    install -m644 internal/desktop/fonts/*-OFL.txt "$out/share/licenses/biternet/"
    wrapProgram "$out/bin/biternet" \
      --prefix PATH : ${lib.makeBinPath [ pkgs.xdg-utils ]} \
      --prefix LD_LIBRARY_PATH : "/run/opengl-driver/lib:${lib.makeLibraryPath desktopLibraries}"
  '';

  meta = {
    description = "BIT campus network client built with Go and MyGo";
    homepage = "https://github.com/bitoolkit/biternet-go";
    license = lib.licenses.unlicense;
    mainProgram = "biternet";
    platforms = lib.platforms.linux;
  };
}
