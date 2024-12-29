# Setup
## Prerequisites
1. [golang](https://go.dev) (I recommend version 1.22.10. Technically, you need 1.21.4 or higher)
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

3. Download hls.js:

```sh
npm i
```

4. Download templ:

```sh
go install github.com/a-h/templ/cmd/templ@latest
```

*You might need to add go binaries to your PATH (add this line to your .bashrc / .zshrc / whatever)*

```sh
export PATH=${PATH}:`go env GOPATH`/bin
```

5. Download regexp2cg:

Not really required, but helps speed up some parts of the code that use regular expressions by generating code for them instead of compiling in runtime.

```sh
go install github.com/dlclark/regexp2cg@main
```

6. Download soundcloakctl:

Soundcloakctl is helper for soundcloak.

```sh
go install git.maid.zone/stuff/soundcloakctl@master
```

7. Download requried JS modules:

```sh
soundcloakctl js download
```

8. Build binary:

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

You can skip some parts if the tools/dependencies have not been updated.
```sh
go get # for go packages

go install github.com/a-h/templ/cmd/templ@latest # templ cli

go install github.com/dlclark/regexp2cg@main # regexp2 codegen cli. not required unless you've installed it

go install git.maid.zone/stuff/soundcloakctl@master # soundcloakctl

soundcloakctl js download # re-download JS modules
```

3. Clean precompressed static files

Those are created by the webserver in order to more efficiently serve static files. They have the `.fiber.gz` extension. You can easily remove them from all directories like this:
```sh
find . -name \*.fiber.gz -type f -delete
```

4. Run codegen and build the binary:
```sh
./build
```

Now, you can run soundcloak with the `./main` binary.

# Contributing
Contributions are appreciated!

We develop soundcloak on [our forgejo instance](https://git.maid.zone/stuff/soundcloak), but feel free to contribute on our [Codeberg](https://codeberg.org/maid-zone/soundcloak) and [Github](https://github.com/maid-zone/soundcloak) as well!

If you want to add a new feature that's not in [the todo list](https://git.maid.zone/stuff/soundcloak/issues/1), please create an issue or discussion first.

If you have updated go dependencies or added new ones, please run `go mod tidy` before commiting.

Any security vulnerabilities should first be disclosed privately to the maintainer ([different ways to contact me are listed here](https://laptopc.at))