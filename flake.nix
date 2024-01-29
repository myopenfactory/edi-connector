{
  description = "myOpenFactory EDI-Client";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem      (system:
        let pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (
              self: super: {
                nsis = super.nsis.overrideAttrs (old: rec {
                  version = "3.09";
                  src =
                      super.fetchurl {
                        url = "mirror://sourceforge/project/nsis/NSIS%203/${version}/nsis-${version}-src.tar.bz2";
                        sha256 = "DNhGxunFkGgCCoe/ylVtTGMPLF1VTBCYAkQlJC3cVuI=";
                      };
                    srcWinDistributable =
                      super.fetchzip {
                        url = "mirror://sourceforge/project/nsis/NSIS%203/${version}/nsis-${version}.zip";
                        sha256 = "Sqaajp405JTmf86Wo+wgCDltyfXUyaGZi87P7hfU4i0=";
                      };

                    srcNsProcessPlugin =
                      super.fetchzip {
                        url = "https://myopenfactory.blob.core.windows.net/static/NsProcess.zip";
                        sha256 = "J3hyBnawhoFTYMAWsEHDVQ0y4HYKecDZsOwd47k1G+Q=";
                        stripRoot = false;
                      };

                    postBuild = ''
                      cp -avr ${srcNsProcessPlugin}/NsProcess/Include/nsProcess.nsh \
                        $out/share/nsis/Include
                      cp -avr ${srcNsProcessPlugin}/NsProcess/Plugin/nsProcessW.dll \
                        $out/share/nsis/Plugins/x86-unicode/nsProcess.dll
                      chmod -R u+w $out/share/nsis
                    '';
                });
              }
            )
          ];
        }; in
        {
          devShells.default = import ./shell.nix { inherit pkgs; };
        }
      );
}
