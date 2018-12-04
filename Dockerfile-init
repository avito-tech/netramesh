FROM debian:stretch

RUN apt-get update && apt-get install -y iptables

COPY iptables-rules.sh /

ENTRYPOINT ["/iptables-rules.sh"]
