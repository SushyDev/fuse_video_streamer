{
	inputs = {
		nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
	};

	outputs = { self, nixpkgs }:
		let
			supportedSystems = nixpkgs.lib.platforms.all;
			devShells = nixpkgs.lib.genAttrs supportedSystems (system: 
				let
					pkgs = import nixpkgs { inherit system; };
					inherit (pkgs) stdenv;

				in 
					{
					default = pkgs.mkShell {
						buildInputs = [
							pkgs.go
							pkgs.protobuf
							pkgs.protoc-gen-go
						];
					};
				}
			);
		in
			{
			inherit devShells;
		};
}
