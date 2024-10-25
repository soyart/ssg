{ pkgs
, version ? "unknown"
, description ? (import ./flake.nix).description
,
}:

pkgs.buildGoModule {
  inherit version;

  pname = "ssg";
  src = ./.;
  vendorHash = "sha256-fxD5o+7uC2lob86TPxlnqT5m7ZYVjIh9ZQANlVb4Pl4=";
  # Go unit tests are already executed by buildGoModule.
  # preBuild would instead be more useful if we want to set Go flags.
  # preBuild = ''
  #   go test ./...;
  # '';

  meta = {
    homepage = "https://github.com/soyart/ssg";
    description = "${description} (go implementation)";
  };
}
