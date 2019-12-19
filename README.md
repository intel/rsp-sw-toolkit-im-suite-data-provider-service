# Data Provider Service
The data provider service provides data to the IMA by polling endpoints, validating
and reshaping data, then pushing it to consuming services (like EdgeX or a Gateway). 

## Design
This service makes use of [goplumber](https://github.com/intel/rsp-sw-toolkit-im-suite-goplumber/),
an ETL framework for configurable, go-template-based pipelines. Most of the
heavy-lifting is handled by that library. In principle, this service supports
anything that one does, but this one comes with some extras specific to the above
use-case.
    
## Service Configuration
This service uses a [configuration.json](app/config/configuration.json) file
for its configuration options:
- port: the HTTP server port for healthchecks
- pipelinesDir: directory from which pipeline definition are loaded
- templatesDir: directory from which templates are loaded
- pipelineNames: list of names of pipelines to load/run from `pipelinesDir`
- customTaskTypes: list of pipelines to use as tasks within other pipelines;
  loaded from the `pipelinesDir`
- mqttClients: list of MQTT client configuration files, also loaded from the
  `pipelinesDir`
- secretsPath: directory from which `secrets` are loaded

### MQTT Clients Configuration
You can configure additional MQTT clients by adding a new `.json` file to the
pipelines directory; it must include an `endpoint` with the MQTT server, but
other config values are optional. Here's an example:

```json
{
  "endpoint": "mosquitto-server:1883",
  "clientID": "data-provider",
  "timeoutSecs": 30,
  "skipCertVerify": true,
  "username": "",
  "password": ""
}
```

Add the `.json` file's name to the list of `mqttClients` in the main configuration. 
The client may then be used in a pipeline by creating a task with `type` set to 
the filename, less the `.json` suffix. For example, `gwMQTT.json` is used in the
`ClusterPipeline.json` with the `sendCommand` task definition:

```json
{
  "type": "gwMQTT",
  "raw": { "name": "rfid/gw/command" },
  "links": { "value": { "from": "rpcMsg" } }
}
```

The `name` is the topic on which to send the `value`.

### Pipeline Configuration
This service loads and runs whichever pipelines are named in the `pipelineNames`
section of the `configuration.json` file. If you wish to add or remove pipelines,
simply update this list.

The service is currently designed to run three pipelines: two provide ASN or SKU 
data to EdgeX, and the other provides cluster configuration to a Gateway. 
To make things easier to manage, common steps (e.g., proxying, schema validation)
are abstracted into smaller pipelines loaded as "custom tasks". 

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

## ANS/SKU Pipelines 
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
- `make mqtt` to start a mosquitto server

