FROM coturn/coturn:4.6-alpine

COPY turnserver.conf /etc/coturn/turnserver.conf
COPY entrypoint.sh /entrypoint.sh

EXPOSE 3478/udp
EXPOSE 3478/tcp
EXPOSE 5349/tcp
EXPOSE 49152-65535/udp

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
