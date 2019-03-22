#!/usr/bin/env bash

NETRA_SIDECAR_PORT=${NETRA_SIDECAR_PORT:-14956}
NETRA_SIDECAR_USER_ID=${NETRA_SIDECAR_USER_ID:-1337}
NETRA_SIDECAR_GROUP_ID=${NETRA_SIDECAR_GROUP_ID:-1337}

INBOUND_INTERCEPT_PORTS=${INBOUND_INTERCEPT_PORTS:-*}
OUTBOUND_INTERCEPT_PORTS=${OUTBOUND_INTERCEPT_PORTS:-*}

NETRA_INBOUND_PROBABILITY=${NETRA_INBOUND_PROBABILITY:-1}
NETRA_OUTBOUND_PROBABILITY=${NETRA_OUTBOUND_PROBABILITY:-1}

function dump {
    iptables-save
}

trap dump EXIT

IFS=,

iptables -t nat -N NETRA_INBOUND
iptables -t nat -N NETRA_OUTBOUND

iptables -t nat -A PREROUTING \
    -m statistic --mode random --probability ${NETRA_INBOUND_PROBABILITY} \
    -j NETRA_INBOUND

iptables -t nat -A OUTPUT \
    -m statistic --mode random --probability ${NETRA_OUTBOUND_PROBABILITY} \
    -j NETRA_OUTBOUND

if [ "${INBOUND_INTERCEPT_PORTS}" == "*" ]; then
    iptables -t nat -A NETRA_INBOUND -p tcp -m tcp -j REDIRECT --to-ports ${NETRA_SIDECAR_PORT}
else
    for port in ${INBOUND_INTERCEPT_PORTS}; do
        iptables -t nat -A NETRA_INBOUND -p tcp -m tcp --dport ${port} -j REDIRECT --to-ports ${NETRA_SIDECAR_PORT}
    done
fi

# avoid loops
iptables -t nat -A NETRA_OUTBOUND -m owner --uid-owner ${NETRA_SIDECAR_USER_ID} -j RETURN
iptables -t nat -A NETRA_OUTBOUND -m owner --gid-owner ${NETRA_SIDECAR_GROUP_ID} -j RETURN
iptables -t nat -A NETRA_OUTBOUND -p tcp -o lo -d 127.0.0.1 -j RETURN

if [ "${OUTBOUND_INTERCEPT_PORTS}" == "*" ]; then
    iptables -t nat -A NETRA_OUTBOUND -p tcp -j REDIRECT --to-ports ${NETRA_SIDECAR_PORT}
else
    for port in ${OUTBOUND_INTERCEPT_PORTS}; do
        iptables -t nat -A NETRA_OUTBOUND -p tcp --dport ${port} -j REDIRECT --to-ports ${NETRA_SIDECAR_PORT}
    done
fi
