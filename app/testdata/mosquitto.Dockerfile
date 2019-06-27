FROM debian:stretch-slim

RUN apt update && apt install -yq --no-install-recommends \
    mosquitto \
    dumb-init \
    && apt clean
WORKDIR /
ADD mosquitto.conf /etc/mosquitto/conf.d/
ENTRYPOINT ["/usr/sbin/mosquitto"]
