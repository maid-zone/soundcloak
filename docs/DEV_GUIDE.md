# Setup
## Prerequisites
1. [golang](https://go.dev) (I recommend version 1.25.1)
2. [git](https://git-scm.com)

## The setup
1. Clone this repository:

```sh
git clone https://git.maid.zone/stuff/soundcloak
```

2. Go into the cloned repository:

```sh
cd soundcloak
```

3. Download required JS modules:

Currently it's just HLS.js

```sh
go tool soundcloakctl js download
```

4. Build binary:

This uses the `build` script, which generates code from templates, generates code for regular expiressions, and then builds the binary.

```sh
./build
```

Now, you can run soundcloak with the `./main` binary. By default, it is listening on `127.0.0.1:4664`. For a configuration guide, [refer to here](INSTANCE_GUIDE.md#configuration-guide)

# Updating your local setup
1. Retrieve the latest code:
```sh
git fetch origin
git pull
```

2. Update dependencies/tools:

```sh
go mod download # update tools and dependencies

go tool soundcloakctl js download # re-download JS modules
```

3. Run codegen and build the binary:
```sh
./build
```

Now, you can run soundcloak with the `./main` binary.

# Contributing
Contributions are appreciated!

We develop soundcloak on [our forgejo instance](https://git.maid.zone/stuff/soundcloak), but feel free to contribute on our [Codeberg](https://codeberg.org/maid-zone/soundcloak) and [Github](https://github.com/maid-zone/soundcloak) as well!

If you want to add a new feature that's not in [the todo list](https://git.maid.zone/stuff/soundcloak/issues/1), please create an issue or discussion first.

If you have updated go dependencies or added new ones, please run `go mod tidy` before commiting.

If you update structs, please run [betteralign](https://github.com/dkorunic/betteralign) to make sure memory layout is optimized.

Any security vulnerabilities should first be disclosed privately to the maintainer ([different ways to contact me are listed here](https://laptopc.at))