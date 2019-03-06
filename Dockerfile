FROM debian:stretch
EXPOSE 9100

RUN apt-get update && \
    apt-get install -y curl wget git ca-certificates && \
    mkdir /app && cd /app && \
    LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/noderig/releases | grep tag_name | head -n 1 | cut -d '"' -f 4) && \
    curl -s https://api.github.com/repos/ovh/noderig/releases | grep ${LAST_RELEASE} | grep browser_download_url | cut -d '"' -f 4 > files && \
    cat files | sort | uniq > filesToDownload && \
    while read f; do wget $f; done < filesToDownload && \
    chmod +x noderig && \
    chown -R nobody:nogroup /app && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/app/noderig"]
