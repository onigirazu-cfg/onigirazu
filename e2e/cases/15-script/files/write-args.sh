#!/bin/bash
printf '%s|%s|%s\n' "$#" "$2" "$(id -u)" > /root/onigirazu-e2e-script
