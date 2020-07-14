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

![Main view](/_docs/main-view.png)