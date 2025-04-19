#!/usr/bin/bash
until false; do
    echo "Respawning.." >&2
    sleep 4
	reset
    go run .
done
