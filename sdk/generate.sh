#!/bin/bash

# Grab the OpenAPI. This can be simplified using a custom command, for
# example see:
# https://github.com/danielgtaylor/huma/blob/main/examples/spec-cmd/main.go
curl http://localhost:8888/openapi-3.0.yaml >openapi-3.0.yaml

# Generate the SDK
oapi-codegen -generate "types,client" -package sdk openapi-3.0.yaml >sdk.go
