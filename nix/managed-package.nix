{
  pkgs,
  package,
  autoStart,
  silentStart,
}:

pkgs.symlinkJoin {
  name = "${package.name}-managed";
  paths = [ package ];
  nativeBuildInputs = [ pkgs.makeWrapper ];
  postBuild = ''
    wrapProgram "$out/bin/biternet" --set BITERNET_NIX_STARTUP ${
      pkgs.lib.escapeShellArg (
        builtins.toJSON {
          auto_start = autoStart;
          silent_start = silentStart;
        }
      )
    }
  '';
  meta = package.meta // {
    mainProgram = "biternet";
  };
}
