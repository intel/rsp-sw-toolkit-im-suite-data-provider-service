# Data Provider Service
The data provider service provides data to the IMA by polling an HTTP endpoint
and pushing it as an EdgeX event.

## Design
This service makes use of [goplumber](https://github.impcloud.net/RSP-Inventory-Suite/goplumber),
an ETL framework for configurable, go-template-based pipelines. Most of the
heavy-lifting is handled by that library. Its actions are controlled by 
configuration. This service defines configuration for this specific use-case
and a few helper functions:
- code:
    - provides a healthcheck endpoint
    - loads pipeline definitions
    - configures template and secret locations 
- config:
    - [service config](#Service-Configuration)
    - ASN/SKU pipeline definitions 
    ([defaults](app/config/pipelines) are included in Docker image)
    - ASN/SKU, EdgeX, and Cloud Connector templates 
    ([defaults](app/config/templates) are included in Docker image)
    - endpoint config and data schemas 
    ([must be provided](#Endpoint-Configuration); 
    examples in [testdata](app/testdata) directory)
    
## Integration Testing
For quick integration testing, this service includes a [Makefile](Makefile) and
[edgex-compose](edgex-compose.yml) file. The compose file brings up EdgeX
services in a manner typical of EdgeX's examples, while the [Makefile](Makefile)
automates steps typically delegated to a stack -- they're set in a Makefile
so to make it easier to run without Docker Swarm. `make up` runs all the
dependencies and then starts an attached instance of the `provider` service.
The [app/testdata](app/testdata) files are bind-mounted into the containers.
`make down` will reverse this.

The data provider service is pulled from the AWS ECR, but you can change it
with something like:

> `REPO=local/data-provider-service TAG=beta make up`

Individual targets:
- `make net` to create a network for the containers
- `make edgex` to just bring up the EdgeX services 
- `make server` to start an `nginx` server for the data files
- `make data-provider` to start the `data-provider` service
- `make cloud-connector` to start the `cloud-connector` service

## Dependencies 
This service needs the following: 
- services:
    - EdgeX core data service instance
    - EdgeX consul service instance
    - Cloud Connector service instance
    - ASN/SKU data server(s)
- config/secrets:
    - ASN/SKU data schemas 
    - Endpoint configuration 

## Service Configuration
This service uses a [configuration.json](app/config/configuration.json) file
for its configuration options:
- port: the HTTP server port for healthchecks
- pipelinesDir: directory from which pipeline definition are loaded
- templatesDir: directory from which templates are loaded
- pipelineNames: list of names of pipelines to load/run from `pipelinesDir`
- secretsPath: directory from which `secrets` are loaded

## Endpoint Configuration
The pipelines load a JSON schema and configuration file from the `secrets` 
directory, and thus they _must_ be provided. Examples are included in the 
[testdata](app/testdata) directory.

The pipeline configurations define:
- siteID: used as a query parameter when `GET`ing the data
- dataEndpoint: the url used for `GET`ing data
- coreDataLookup: url for looking up the core-data service in EdgeX's consul instance
- cloudConnEndpoint: the url for the Cluod Connector
- dataSchemaFile: the name of the secret file with the JSON schema.
- oauthConfig:
    - useAuth: if true, auth data is sent to the Cloud Connector
    - oauthEndpoint: the endpoint the Cloud Connector should use for tokens
    - oauthCredentials: `username:password` the Cloud Connector should use

## ANS/SKU Pipelines and Templates
The pipelines and templates are included in the service image, but can be
overridden via Docker volumes, secrets, or configs. Both pipelines use an 
`interval` trigger and have essentially the same flow:

- Load a config secret for some pipeline-specific options.
- Issue a `GET` to the EdgeX `consul` instance to find the Core Data url.
- Get the timestamp for when the data was last updated. 
- Construct a proxy request to `GET` the data. 
- Send the proxy request to the Cloud Connector.
- Extract the Cloud Connector's response. 
- Load a JSON schema.
- Validate the incoming data against the schema.
- Construct an EdgeX event
  - the `device` for the event is `ASN_Data_Device` or `SKU_Data_Device`
  - the event contains a single reading
  - the `name` of the reading is `ASN_data` or `SKU_data`
  - the `value` of the reading is the base64 encoded data
- Send the EdgeX event to the Core Data URL.
- Update the timestamp for when the data was last updated.

## Templates
There are three template namespaces included in the service. They can be overridden
with Docker volumes/secrets/configs if desired. They include the following: 
- ASNSKU:
    - ccDest: construct the URL from which data is pulled
    - edgeXReadings: construct the EdgeX reading for the EdgeX event
- cloudConn:
    - proxyCCRequest: construct the Cloud Connector request body
    - extractCCResponse: check and extract the response
- edgex:
    - edgeXEvent: construct the EdgeX event body
    - createEdgeXURL: construct a URL for an EdgeX API request 


