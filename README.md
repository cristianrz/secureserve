# SecureServe

SecureServe sets up a TLS-enabled file browser server using `filebrowser`. It
generates a random password, creates a TLS certificate with `mkcert`, and
starts the server with the specified directory, protecting it with the
generated password.

## Prerequisites

- `mkcert`
- `filebrowser`

## Usage

Set the DIRECTORY environment variable (defaults to current directory) and run the script:

```sh
DIRECTORY=<your_directory> ./secureserve.sh
```

Example Output

```json
{
  "directory": "/path/to/your/directory",
  "url": [
    "https://192.168.1.2:8081",
    "https://localhost:8081",
    "https://yourhostname:8081"
  ],
  "username": "user",
  "password": "randompassword"
}
```

## License

MIT License.

