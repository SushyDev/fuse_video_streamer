{
	description = "Build dependencies";

	inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

	outputs = { self, nixpkgs }:
		let
			supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

			mkDepsBundle = (system:
				let
					pkgs = import nixpkgs { inherit system; };
				in
				pkgs.stdenv.mkDerivation {
					name = "dependencies-bundle";
					dontUnpack = true;

					nativeBuildInputs = [
						pkgs.pkgsStatic.fuse
						pkgs.cacert
					];

					installPhase = ''
						mkdir -p $out/bin $out/etc/ssl/certs
						cp ${pkgs.pkgsStatic.fuse}/bin/fusermount $out/bin/
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
