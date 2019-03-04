FROM debian:stretch
EXPOSE 9100

ENV NODERIG_VERSION=v2.3.1

RUN apt-get update && \
    apt-get install -y wget && \
    wget -q https://github.com/ovh/noderig/releases/download/$NODERIG_VERSION/noderig && \
    chmod +x noderig

ADD custom_collectors /collectors

ENTRYPOINT ["/noderig"]
