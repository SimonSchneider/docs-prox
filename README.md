# Docs Prox

Dynamically cloud native OpenAPI documentation proxy and swagger portal.

Why would I use Docs Prox? You have a lot of microservices that all expose OpenAPI 
documentation but you do not have a single place to find them all. Docs Prox aggregates
the OpenAPI documentations and lets you dynamically add and remove services with minimal
work. (Service discovery through Kubernetes and other sources available). This makes it
easier to understand and find what services and APIs are available with minimal effort.

Docs Prox is however not a developer portal as of now. It is a dynamic aggregator of OpenAPI
specifications.

## In more detail

It is an OpenAPI (Swagger) aggregator. At the core there is a server-side go application
which can be configured to proxy OpenAPI specifications from multiple different sources
and expose them through it's API. The configurations range from static environment variables,
dynamic directory and file watchers to dynamic kubernetes service discovery and configuration.

The web-ui is a small wrapper of swagger-ui which leverages the server-side to add
a sidebar in which we can switch between the available OpenAPI specifications.
The web-ui allows users to pin services in the sidebar to help with organization.

![Main view](/docs/main-view.png)

# Demo

http://34.73.146.17/

Sample APIs taken from
https://apis.guru/browse-apis/

## Configuration

There are currently 3 different docs-discovery-providers. Each key/name
(the name in the sidebar of the UI) must be globally unique and whatever
provider is first to register that name is the owner of it.

### Environment Provider
Looks for env variables with a configurable prefix and adds them 
assuming the content is a URL pointing at the openAPI documentation
ie. prefix is `SWAGGER_` and an env variable
`export SWAGGER_TEST_1=http://test1.com/openapi` will configure a entry in the
UI with name `test-1` proxying the openAPI spec at `http://test1.com/openapi`

### File Provider
Looks for files in a configurable directory, the files should have a configurable
prefix and one of two file-extensions denoting the two supported file types.

The directory will be watched for changes and any updates to existing files,
removing of files or adding of new files will be immediately reflected in the UI.

#### Json
Files with extension `.json` should contain the json openAPI specification

#### Url
Files with extension `.url` should contain one `name: url` pair per row.
They will be added to the UI with service name `$name` and proxy the URL `$url`.
ie.
```
service 123: http://service123.com/openapi
another service: http://another-service.com/openapi
```

### Kubernetes Provider
Watches a kubernetes cluster for two types of resources.

#### Service

#### ConfigMap
