{
	description = "Build dependencies";

	inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

	outputs = { self, nixpkgs }:
		let
		supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

		mkDepsBundle = (system:
			let
			pkgs = import nixpkgs { inherit system; };
			pkgsMusl = import nixpkgs {
				inherit system;
				crossSystem = {
					config = "${pkgs.stdenv.targetPlatform.parsed.cpu.name}-unknown-linux-musl";
					isStatic = true;
				};
			};

			in
			pkgs.stdenv.mkDerivation {
				name = "dependencies-bundle";
				dontUnpack = true;

				# We only need the statically built fuse and cacert.
				nativeBuildInputs = [
					pkgs.pkgsStatic.fuse
					pkgs.cacert
				];

				# With a truly static binary, the install phase is minimal.
				installPhase = ''
					mkdir -p $out/bin $out/etc/ssl/certs

					# 1. Copy the single, self-contained fusermount binary.
					cp ${pkgs.pkgsStatic.fuse}/bin/fusermount $out/bin/

					# No /lib directory or linker is needed.

					# 2. Copy the CA certificate bundle.
					cp -L ${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt $out/etc/ssl/certs/ca-certificates.crt
				'';
			}
		);
		in
		{
			packages = nixpkgs.lib.genAttrs supportedSystems (system: {
				default = mkDepsBundle system;
			});
		};
}
