#!/bin/sh

echo 'os.uptime' `date +%s%N | cut -b1-10` `awk '{print $1}' /proc/uptime`