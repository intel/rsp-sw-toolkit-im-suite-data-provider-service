FROM scratch
ADD rfid-mqtt-provider-service /
WORKDIR /

EXPOSE 8080
ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT
HEALTHCHECK --interval=5s --timeout=3s CMD [ "/rfid-mqtt-provider-service","-isHealthy" ]
ENTRYPOINT ["/rfid-mqtt-provider-service"]
