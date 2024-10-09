rec {
  description = "Static site generator from romanzolotarev.com";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      homepage = "https://romanzolotarev.com/ssg.html";

      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
      version = builtins.substring 0 8 lastModifiedDate;

      # The set of systems to provide outputs for
      allSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      # A function that provides a system-specific Nixpkgs for the desired systems
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in

    {
      packages = forAllSystems ({ pkgs }: {
        default = pkgs.writeShellApplication {
          name = "ssg";

          runtimeInputs = with pkgs; [
            coreutils
            lowdown
          ];

          text = ''
            ${builtins.readFile ./ssg.sh}
          '';

          meta = {
            inherit description homepage;
          };
        };

        impure = pkgs.stdenv.mkDerivation {
          inherit version;
          pname = "ssg";

          src = ./.;
          installPhase = ''
            mkdir -p $out/bin;
            cp ssg.sh $out/bin/ssg;
          '';

          meta = {
            inherit homepage;
            description = "${description} (impure version)";
          };
        };

        ssg-go = pkgs.buildGoModule {
          inherit version;

          pname = "ssg";
          src = ./.;
          vendorHash = "sha256-fxD5o+7uC2lob86TPxlnqT5m7ZYVjIh9ZQANlVb4Pl4=";

          meta = {
            homepage = "https://github.com/soyart/ssg";
            description = "${description} (go implementation)";
          };
        };
      });

      devShells = forAllSystems ({ pkgs }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            nixd
            nixpkgs-fmt

            bash-language-server
            shellcheck
            shfmt

            coreutils
            lowdown

            go
            gopls
            gotools
            go-tools
          ];
        };
      });
    };
}
