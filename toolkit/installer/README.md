# Quickstart installation guide

Note, this is work in progress.  The guide will become saner as the installer
evolves.

Download as:

```
wget https://storage.googleapis.com/stolos-dev/release/v0.2.0-26-g5ef2be0-dirty/install-linux-amd64.tar.gz
```

Unpack as:
```
tar xzvvf install-linux-amd64.tar.gz

```

Then change into the directory, and edit the file `configs/example.json` to
fit your settings.

From the installer home directory run:

```
./installer --config=configs/example.json
```
