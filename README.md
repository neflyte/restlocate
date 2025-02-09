# `restlocate`

A trivial HTTP wrapper around the `locate` command that allows for basic pattern and regex searches with optional case-insensitivity.

## Build

To build, run:

```
go build
```

## Usage

```
./restlocate [--port 8080] [--address ""]
```

Default port is `8080`. Default address is the empty string; same effect as `0.0.0.0`

## Query

The service can be queried using an URL like the following:

```
http://localhost:8080/locate?[search=xxxxx|regex=xxxxx][&ci=true]
```

For example, query the basic pattern `foo` with case-insensitivity:

```
http://localhost:8080/locate?search=foo&ci=true
```

A JSON list of search results will be returned.
