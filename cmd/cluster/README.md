# CLUSTER

flatten direcotories into single file wihtout compression

> empty directories are lost

## building

```bash
git clone https://github.com/velox0/pocketutils.git
cd pocketutils
make cluster
```

## usage

### flatten directory to `.cluster`

produce a `<dirname>.cluser` file

```bash
path/to/cluster path/to/directory
```

### retreive back files from `.cluster`

revert back from `<dirname>.cluser` file to the filesystem

```bash
path/to/cluster path/to/directory.cluster
```
