# FlowServer
**FlowServer** is a Go library designed to simplify the creation of robust and scalable HTTP servers using the [Gin framework](https://github.com/gin-gonic/gin). It integrates Swagger for API documentation, logging with Gin middleware, and makes server initialization effortless.

---

## Features
- **Effortless Server Initialization**: Quickly bootstrap a Gin-based HTTP server with minimal configuration.
- **Swagger Integration**: Automatically generates API documentation using `swag` based on annotations in your code.
- **Logging Middleware**: Provides request logging with IP anonymization for GDPR compliance.
- **Dynamic Configuration**: Supports environment-based configuration for production and development.
- **Asynchronous Server Start**: Easily integrate with other concurrent components in your application.

---

## Installation
Import in other projects:
```commandline
export GOPRIVATE=github.com/LucaSchmitz2003/*
GIT_SSH_COMMAND="ssh -v" go get github.com/LucaSchmitz2003/FlowServer@main
```

---

### Usage
Server Initialization:

```go
ctx := context.Background()

// Initialize the server
address, router := apiHelper.InitServer(ctx, defineRoutes)

// Start the server in a goroutine
keepAlive := apiHelper.StartServer(ctx, router, address)
defer keepAlive()
```

## Licenses

This project is licensed under the MIT License. It uses third-party libraries that are licensed under the following terms (direct dependencies only):

- BSD-3-Clause License
- Apache License 2.0

Please refer to the respective license texts of these libraries, which are referenced in `go.mod`. By using this project, you agree to comply with the license terms of these third-party dependencies.