#!/bin/bash

function getCurrentUnixTimestamp {
	printf %s $(date +%s)
}

echo " (i) Generating current timestamp..."
current_unix_timestamp=$(getCurrentUnixTimestamp)
echo "     Timestamp: $current_unix_timestamp"
echo " (i) Done"