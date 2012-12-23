[![Build Status](https://travis-ci.org/robertkowalski/graylog-golang.png?branch=master)](https://travis-ci.org/robertkowalski/graylog-golang)

# graylog-golang

## graylog-golang is a full implementation for sending messages in GELF (Graylog Extended Log Format) from Google Go (Golang) to Graylog


# Example

```go
package main

import (
  "github.com/robertkowalski/graylog-golang"
)

func main() {

  g := gelf.New(gelf.Config{})

  g.Log(`{
      "version": "1.0",
      "host": "localhost",
      "timestamp": 56765675675,
      "facility": "Google Go",
      "short_message": "Hello From Golang! PACKAGE TEST"
  }`)
}
```

# Setting Config Values

```go
g := New(Config{
  GraylogPort:     80,
  GraylogHostname: "example.com",
  Connection:      "wan",
  MaxChunkSizeWan: 42,
  MaxChunkSizeLan: 1337,
})
```

# Tests
```
go test
```
