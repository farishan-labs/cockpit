#!/bin/sh
set -eu
cd "$(dirname "$0")"
swiftc -O cockpit-icon.swift -o cockpit-icon
echo "built menubar/cockpit-icon"