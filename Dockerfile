FROM scratch
COPY data-provider-service /

COPY app/config/templates /templates

EXPOSE 8080
ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT
HEALTHCHECK --interval=5s --timeout=3s CMD [ "/data-provider-service","-isHealthy" ]
ENTRYPOINT ["/data-provider-service"]
