<p align="center">
  <img src="assets/logo.svg" width="80" height="80" alt="krypt">
</p>

<h1 align="center">krypt</h1>

<p align="center">
  Zero-knowledge encrypted pastebin CLI. All encryption runs locally — the server never sees your data.
</p>

---

## How it works

1. Content is encrypted locally with AES-256-GCM. The key is derived via Argon2id.
2. Only the ciphertext is sent to the server.
3. The decryption key lives in the URL fragment (`#...`), which is never sent over the network.

## Install

Download a binary from [Releases](../../releases), or build from source:

```
go install krypt@latest
```

## Usage

### Create a paste

```
echo "secret" | krypt paste
```

```
krypt paste --title "My Script" --lang bash script.sh
```

### Options

```
--server    Server URL (overrides config and KRYPT_SERVER env)
--title     Encrypted title
--lang      Language hint (go, python, bash, etc.)
--password  Password for double-layer encryption
--expires   Expiry duration: 10m, 1h, 7d, 30d
--burn      Delete after first read
```

### Retrieve a paste

```
krypt get 'https://krypt.li/p/abc123#key'
```

```
krypt get --password hunter2 'https://krypt.li/p/abc123#key'
```

Pass `--meta` to print title, language, read count, and expiry to stderr.

### Configuration

```
krypt config set server https://krypt.li
krypt config get server
krypt config path
```

Server resolution order: `--server` flag > `KRYPT_SERVER` env > config file > built-in default.

On first run, `krypt` will prompt you for your server URL.

## Build from source

```
go build -o krypt .
```

## License

MIT
