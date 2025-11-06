# VOID

Eats command stdout and stderr

> You can just `<command> > /dev/null 2>&1` but this is okay too and works in all shells

## example

```bash
path/to/void <command> <args>
```

![example](example.gif)

## building

```bash
git clone https://github.com/velox0/pocketutils.git
cd pocketutils
mkdir build
go build -o ./build ./cmd/void
```

## installation

```bash
sudo mv ./build/void /usr/bin/
```
