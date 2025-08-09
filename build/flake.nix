# build/flake.nix
{
	description = "A flake that provides a minimal, musl-based FUSE environment";

	inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

	outputs = { self, nixpkgs }:
		let
		# List of architectures we want to support.
		supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

		# A function that generates the dependency bundle for a given system.
		mkDepsBundle = (system:
			let
			pkgs = import nixpkgs { inherit system; };
			pkgsMusl = import nixpkgs {
				inherit system;
				crossSystem = {
					config = "${pkgs.stdenv.targetPlatform.parsed.cpu.name}-unknown-linux-musl";
				};
			};
			in
			# Use buildEnv, which is the correct tool for bundling existing packages
			# without compiling from source.
			pkgs.stdenv.mkDerivation {
				name = "dependencies-bundle";

				dontUnpack = true;

				# List the packages whose contents we want to be made available.
				# buildEnv will automatically create a directory structure (/bin, /lib, etc.)
				# and symlink the contents of these packages into it.
				paths = [
					pkgsMusl.fuse
					pkgs.cacert
				];

				installPhase = ''
					mkdir -p $out/bin $out/lib $out/etc/ssl/certs

					# 1. Copy the musl-based fusermount and its libs for the target architecture.
					cp ${pkgsMusl.fuse}/bin/fusermount $out/bin/
					#cp -a ${pkgsMusl.fuse}/lib/. $out/lib/

					# 2. Copy the CA certificate bundle, dereferencing the symlink from the cacert package.
					cp -L ${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt $out/etc/ssl/certs/ca-certificates.crt
				'';
			}
		);
		in
		{
		# Generate the default package for each supported system.
		packages = nixpkgs.lib.genAttrs supportedSystems (system: {
			default = mkDepsBundle system;
		});
	};
}
