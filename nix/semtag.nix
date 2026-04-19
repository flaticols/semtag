# This file is auto-updated by GoReleaser on each release.
# Do not edit manually — changes will be overwritten.
{ lib, stdenvNoCC, fetchurl }:
let
  version = "0.0.1";
  tarballs = {
    "aarch64-darwin" = fetchurl {
      url = "https://github.com/flaticols/semtag/releases/download/v${version}/semtag_Darwin_arm64.tar.gz";
      hash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # arm — updated by GoReleaser
    };
    "x86_64-darwin" = fetchurl {
      url = "https://github.com/flaticols/semtag/releases/download/v${version}/semtag_Darwin_x86_64.tar.gz";
      hash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # x86 — updated by GoReleaser
    };
  };
in
stdenvNoCC.mkDerivation {
  pname = "semtag";
  inherit version;

  src = tarballs.${stdenvNoCC.hostPlatform.system}
    or (throw "Unsupported system: ${stdenvNoCC.hostPlatform.system}. semtag ships macOS binaries only.");

  unpackPhase = "tar xzf $src";

  installPhase = ''
    runHook preInstall
    install -Dm755 semtag $out/bin/semtag
    runHook postInstall
  '';

  meta = {
    description = "Semantic versioning CLI for Git and Jujutsu";
    homepage = "https://github.com/flaticols/semtag";
    license = lib.licenses.mit;
    mainProgram = "semtag";
    platforms = [ "aarch64-darwin" "x86_64-darwin" ];
  };
}
