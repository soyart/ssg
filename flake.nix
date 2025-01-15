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

          text = builtins.readFile ./ssg.sh;

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
          src = ./ssg-go;
          vendorHash = "sha256-89MtPLdBD0lF7YOrhMgSB0q0AdKylBAiLmPQayL+M9I=";

          buildPhase = ''
            echo "Note: only building ./cmd/ssg for ssg-go"
            mkdir -p bin $out/bin
            go build -o ./bin/ssg ./cmd/ssg
            mv ./bin/ssg $out/bin/
          '';

          # Go unit tests are already executed by buildGoModule.
          # preBuild would instead be more useful if we want to set Go flags.
          # preBuild = ''
          #   go test ./...;
          # '';

          meta = {
            homepage = "https://github.com/soyart/ssg";
            description = "${description} (go implementation)";
          };
        };

        soyweb = pkgs.buildGoModule {
          inherit version;

          pname = "soyweb";
          src = ./soyweb;
          vendorHash = "sha256-Rc/rZ0AAa+Fyanhn5OHa8g3N9UMK0u1xy2JxjbaQQDs=";
          meta = {
            homepage = "https://github.com/soyart/ssg";
            description = "soyweb - ssg wrapper";
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
