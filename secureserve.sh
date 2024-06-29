#!/bin/sh

# LICENSE: MIT License
# Short explanation: This script generates a password, sets up a TLS certificate using mkcert,
# and starts a filebrowser server with the specified directory, protecting it with the generated password.

# Set options
set -eu

# Global definitions
CERT_DIR="$HOME/.local/share/secureserve"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"
DB_PATH="$CERT_DIR/filebrowser.db"
PORT=8081
USERNAME="user"

# _log_err message
#
# _log_err prints the error message to stderr.
_log_err()
{
    printf '%s: %s\n' "$(basename "$0")" "$*" >&2
}

# _generate_password
#
# _generate_password generates a password composed of 3 random words.
_generate_password()
(
  grep -v \' /usr/share/dict/words | shuf -n3 | tr '\n' ' ' | sed 's/ $//g'
)

# generate_certificate
#
# generate_certificate generates a TLS certificate using mkcert if it does not already exist.
generate_certificate()
{
  mkdir -p "$CERT_DIR"

  if [ ! -f "$CERT_FILE" ] || [ ! -f "$KEY_FILE" ]; then
    mkcert -cert-file "$CERT_FILE" -key-file "$KEY_FILE" localhost
  fi
}

# _hash_password password
#
# _hash_password hashes the given password using filebrowser hash.
_hash_password()
(
  password="$1"
  hashed_password=$(filebrowser hash "$password")
  printf '%s\n' "$hashed_password"
)

# start_filebrowser password hashed_password
#
# start_filebrowser starts the filebrowser server with the given password and hashed password.
start_filebrowser()
{
  directory="${DIRECTORY:-$PWD}"
  password="$1"
  hashed_password="$2"

  # Remove old database if it exists
  rm -f "$DB_PATH"

  # Initialize filebrowser configuration
  filebrowser config init --database "$DB_PATH" >/dev/null 2>&1

  # Add user to filebrowser
  filebrowser users add "$USERNAME" "$password" --perm.admin --database "$DB_PATH" >/dev/null 2>&1

  # Gather IP addresses
  ip_addresses=$(hostname -I | tr ' ' '\n' | sed 's/$/:'"$PORT"'/')
  hostname_f=$(hostname -f)

  # Create JSON output
  json_output=$(printf '{\n  "directory": "%s",\n  "url": [\n' "$directory")
  for ip in $ip_addresses; do
    json_output=$(printf '%s    "https://%s",\n' "$json_output" "$ip")
  done
  json_output=$(printf '%s    "https://localhost:%s",\n    "https://%s:%s"\n  ],\n  "username": "%s",\n  "password": "%s"\n}\n' "$json_output" "$PORT" "$hostname_f" "$PORT" "$USERNAME" "$password")

  # Output JSON to stdout
  printf '%s\n' "$json_output"

  # Start filebrowser
  exec filebrowser --database "$DB_PATH" --address "0.0.0.0" --port "$PORT" --root "$directory" --cert "$CERT_FILE" --key "$KEY_FILE" >/dev/null 2>&1
}

# usage
#
# usage prints the usage information for this script.
usage()
{
  printf "Usage: DIRECTORY=<directory> %s\n" "$(basename "$0")"
  printf "\nEnvironment Variables:\n"
  printf "  DIRECTORY   Specify the directory to serve (default is current directory)\n"
  exit 1
}

# Main code
if [ -z "${DIRECTORY:-}" ]; then
  DIRECTORY="$PWD"
fi

password=$(_generate_password)
generate_certificate
hashed_password=$(_hash_password "$password")
start_filebrowser "$password" "$hashed_password"

