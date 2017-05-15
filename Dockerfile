FROM alpine:3.5
EXPOSE 9100
ENTRYPOINT ["/usr/bin/noderig"]
COPY noderig /usr/bin/noderig
