#!/usr/bin/env bash
iptables -t nat -A PREROUTING -p tcp -m tcp -j REDIRECT --to-ports 14956
iptables -t nat -A OUTPUT -m owner --uid-owner 1337 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 1337 -j RETURN
iptables -t nat -A OUTPUT -p tcp -o lo -d 127.0.0.1 -j RETURN
iptables -t nat -A OUTPUT -p tcp -j REDIRECT --to-ports 14956
