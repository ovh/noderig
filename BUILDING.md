# Building Noderig

Noderig need `dep` and `golangci-lint` go package to correctly be build. 
If you do not have them you can execute 

```sh
make install
```

## Build dev Noderig version

Otherwise to build a dev version of Noderig, you can just execute:

```sh
make deps  # To get Noderig dependancies
make lint # To apply golangci-lint linter
make dev  # Build binary
```

You will now be able to find Noderig binary in `./build` folder.

Noderig is executable with:

```sh
./build/noderig
```

## Build release Noderig version

To build Noderig release version, you can follow previous steps and then build Noderig with:

```sh
make release
```

## Reload Noderig dependancies

To reload all Noderig dependancies you can execute: 

```sh
make init
```