{
  description = "Semantic versioning CLI for Git and Jujutsu";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      packages = forAllSystems (pkgs: {
        default = pkgs.buildGoModule {
          pname = "semtag";
          version = "0.0.1";
          src = ./.;
          go = pkgs.go;
          # vendor/ directory is included in the source tree
          vendorHash = null;
          doCheck = false;
          meta = with pkgs.lib; {
            description = "Semantic versioning CLI for Git and Jujutsu";
            homepage = "https://github.com/flaticols/semtag";
            license = licenses.mit;
            mainProgram = "semtag";
            platforms = platforms.darwin;
          };
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools ];
        };
      });
    };
}
