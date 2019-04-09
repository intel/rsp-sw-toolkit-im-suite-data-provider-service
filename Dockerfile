FROM scratch
ADD data-provider-service /

EXPOSE 8080
ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT
ENTRYPOINT ["/data-provider-service"]