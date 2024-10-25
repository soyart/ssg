{ pkgs ? let
    lock = (builtins.fromJSON (builtins.readFile ./flake.lock)).nodes.nixpkgs.locked;
    nixpkgs = fetchTarball {
      url = "https://github.com/nixos/nixpkgs/archive/${lock.rev}.tar.gz";
      sha256 = lock.narHash;
    };
  in
  import nixpkgs { }
}: {
  default = pkgs.pkgsCross.aarch64-darwin.callPackage ./ssg-go.nix {
    inherit pkgs;
  };

  amd64 = pkgs.pkgsCross.x86_64-darwin.callPackage ./ssg-go.nix {
    inherit pkgs;
  };
}
