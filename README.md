# Huma Demo

This is a demo of the [Huma](https://github.com/danielgtaylor/huma) framework. Huma is a Go library for building web APIs with OpenAPI 3 that are easy to use, easy to understand, and easy to maintain.

The purpose of this demo is to show a simple API, so you won't see a real, complex project with all the bells & whistles and code organization you would expect in a real project. Instead, you'll see a simple API with a few endpoints and some basic functionality, but more than a "hello world" so that we can show off some of the more interesting features built-in to Huma.

## Getting Started

Just check out the project and then you can run it:

```bash
# Run the server.
$ go run .
```

> Alternatively you can run via [Air](https://github.com/cosmtrek/air) for hot reloading.

You should see the docs at http://localhost:8888/docs. Use them to create a test channel!

Then you can use Restish or curl to call the API:

```bash
# One-time setup
$ brew install danielgtaylor/restish/restish
$ restish api configure demo http://localhost:8888

# List available commands
$ restish demo --help

# List channels (via operation ID or URL)
$ restish demo list-channels
$ restish demo/channels

# Edit the channel you created via your editor. VSCode provides linting &
# completion as you type.
$ restish edit -i demo/channels/my-test-channel
```

## Unit Testing

You can run the unit tests for the API, which are in the `main_test.go` file:

```bash
# Run tests with coverage info
$ go test -cover
```

## SDK Generation

You can also generate an SDK for the API (note the API must be running in another tab):

```bash
# One-time setup
$ go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest

# Generate the SDK
$ cd sdk
$ ./generate.sh
```

Then you can run the demo project using the generated SDK:

```bash
# Run the demo project
$ go run ./example
```

## Performance Testing

You can do a quick perfomance test using `wrk`:

```bash
# One-time setup
$ brew install wrk

# Ensure the API is running & you have a channel called `test`, then run:
$ wrk -c 20 -d 10s -t 10 http://localhost:8888/channels/test
```

During the test you can run `htop` and use F4 to filter to `go run .`. Note that you may also want to disable printing requests in the middleware for the perf test.

## Things to Demo

- [ ] Go structs for models
- [ ] Validation tags & resolvers
- [ ] Composable/shareable input parameters
- [ ] Registering operations
- [ ] API setup, extra docs
- [ ] Middleware example
- [ ] Unit testing the API
- [ ] Interactive API docs
- [ ] Restish to call the API, tab completion
- [ ] Exhaustive errors built-in
- [ ] Conditional updates
- [ ] Live editing of resources with validation
- [ ] SDK generation & example usage
- [ ] Performance test

Things that could be added to the demo as an exercise:

- [ ] More formats (e.g. CBOR)
- [ ] Auto `PATCH`
- [ ] Server Sent Events
- [ ] CLI commands & arguments
