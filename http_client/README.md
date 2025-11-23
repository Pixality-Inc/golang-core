# http_client

HTTP client based on [fasthttp](https://github.com/valyala/fasthttp).

## Quick Start

```go
import (
    "context"
    "time"
    
    "github.com/pixality-inc/golang-core/http_client"
    "github.com/pixality-inc/golang-core/logger"
)

// create config
config := &http_client.ConfigYaml{
    BaseUrlValue: "https://api.example.com",
    TimeoutValue: 30 * time.Second,
}

// create client
log := logger.NewLoggableImplWithService("my-service")
client := http_client.NewClientImpl(log, config)

// make request
resp, err := client.Get(context.Background(), "/users",
    http_client.WithQueryParam("limit", "10"),
)
```

## Configuration

### Basic Configuration

```go
config := &http_client.ConfigYaml{
    BaseUrlValue:            "https://api.example.com",
    TimeoutValue:            30 * time.Second,
    InsecureSkipVerifyValue: false,
    UseRequestIdValue:       true,
}
```

### Advanced Configuration

```go
config := &http_client.ConfigYaml{
    // basic settings
    BaseUrlValue:    "https://api.example.com",
    TimeoutValue:    30 * time.Second,
    NameValue:       "my-api-client",
    
    // connection pool settings
    MaxConnsPerHostValue:     512,
    MaxIdleConnDurationValue: 90 * time.Second,
    ReadTimeoutValue:         30 * time.Second,
    WriteTimeoutValue:        30 * time.Second,
    MaxConnWaitTimeoutValue:  5 * time.Second,
    
    // base headers for all requests
    BaseHeadersValue: http_client.Headers{
        "User-Agent": []string{"my-service/1.0"},
        "Accept":     []string{"application/json"},
    },
    
    // retry policy
    RetryPolicyValue: &http_client.RetryPolicy{
        MaxAttempts:        3,
        InitialInterval:    100 * time.Millisecond,
        BackoffCoefficient: 2.0,
        MaxInterval:        5 * time.Second,
    },
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `BaseUrl` | string | "" | Base URL for all requests |
| `Timeout` | duration | 0 | Request timeout |
| `Name` | string | "http_client" | Client name for logs |
| `InsecureSkipVerify` | bool | false | Skip TLS verification |
| `UseRequestId` | bool | false | Include X-Request-Id header from context |
| `MaxConnsPerHost` | int | 512 | Max connections per host |
| `MaxIdleConnDuration` | duration | 90s | Idle connection lifetime |
| `ReadTimeout` | duration | Timeout | Read timeout (uses Timeout if 0) |
| `WriteTimeout` | duration | Timeout | Write timeout (uses Timeout if 0) |
| `MaxConnWaitTimeout` | duration | 0 | Max time to wait for connection |
| `BaseHeaders` | Headers | nil | Headers for all requests |
| `RetryPolicy` | *RetryPolicy | nil | Retry configuration |

## Usage Examples

### GET Request

```go
// simple GET
resp, err := client.Get(ctx, "/users")

// with query parameters
resp, err := client.Get(ctx, "/users",
    http_client.WithQueryParam("limit", "10"),
    http_client.WithQueryParam("offset", "20"),
)

// with multiple params
resp, err := client.Get(ctx, "/users",
    http_client.WithQueryParams(http_client.QueryParams{
        "limit":  "10",
        "offset": "20",
        "sort":   "name",
    }),
)

// with headers
resp, err := client.Get(ctx, "/protected",
    http_client.WithHeader("Authorization", "Bearer "+token),
)
```

### POST Request

```go
// with raw body
resp, err := client.Post(ctx, "/users",
    http_client.WithBody([]byte(`{"name":"John"}`)),
)

// with JSON body (automatic marshaling)
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

resp, err := client.Post(ctx, "/users",
    http_client.WithJsonBody(User{Name: "John", Age: 30}),
)

// with headers
resp, err := client.Post(ctx, "/users",
    http_client.WithJsonBody(user),
    http_client.WithHeader("Authorization", "Bearer "+token),
    http_client.WithHeader("X-Idempotency-Key", uuid),
)
```

### PUT, PATCH, DELETE

```go
// PUT
resp, err := client.Put(ctx, "/users/123",
    http_client.WithJsonBody(updatedUser),
)

// PATCH
resp, err := client.Patch(ctx, "/users/123",
    http_client.WithBody([]byte(`{"name":"Jane"}`)),
)

// DELETE
resp, err := client.Delete(ctx, "/users/123")
```

### Multipart Form Data

#### Standard Usage (Recommended)

```go
// create form data
formData := http_client.NewFormData()
formData.AddField("name", "photo.jpg")
formData.AddField("description", "My photo")
formData.AddFile("file", "photo.jpg", "image/jpeg", fileReader)

// upload via http_client
resp, err := client.Post(ctx, "/upload",
    http_client.WithFormData(formData),
)
```

#### Advanced Usage with swagger-generated clients

If using swagger-generated clients that require direct body/contentType access:

```go
// create form data
formData := http_client.NewFormData()
formData.AddFile("file", "photo.jpg", "image/jpeg", fileReader)

// build to get body and content type
body, contentType, err := formData.Build()
if err != nil {
    return err
}

// use with swagger-generated client
response, err := swaggerClient.UploadWithBodyWithResponse(
    ctx,
    params,
    contentType,  // from Build()
    body,         // from Build()
)
```

### Custom HTTP Methods

```go

// any custom method
resp, err := client.Do(ctx, "CUSTOM", "/endpoint",
    http_client.WithBody(data),
)
```

### Combining Options

```go
resp, err := client.Post(ctx, "/api/users",
    http_client.WithJsonBody(user),
    http_client.WithHeader("Authorization", "Bearer "+token),
    http_client.WithHeader("X-Request-ID", requestID),
    http_client.WithQueryParam("notify", "true"),
)
```

## Request Options

All request options can be combined in any order:

| Option | Description |
|--------|-------------|
| `WithBody([]byte)` | Set raw request body |
| `WithJsonBody(any)` | Set JSON-encoded body (automatic marshal) |
| `WithFormData(FormDataInterface)` | Set multipart form data |
| `WithHeader(key, value)` | Add single header |
| `WithHeaders(Headers)` | Add multiple headers |
| `WithQueryParam(key, value)` | Add single query parameter |
| `WithQueryParams(QueryParams)` | Add multiple query parameters |

## Response Handling

### Standard Response

```go
type Response struct {
    StatusCode int
    Headers    Headers
    Body       []byte
}

resp, err := client.Get(ctx, "/users")
if err != nil {
    // handle error
}

fmt.Println(resp.StatusCode)
fmt.Println(string(resp.Body))
fmt.Println(resp.Headers["Content-Type"])
```

### Typed Response

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

resp, err := client.Get(ctx, "/users/123")
if err != nil {
    return err
}

typed, err := http_client.AsJson(resp, User{})
if err != nil {
    return err
}

fmt.Println(typed.Entity.Name)
```


## FAQ

### When should I use `Build()` method?

The `Build()` method is needed in **specific cases** when working with external libraries that require direct access to form data body and content type:

**Use `WithFormData()` (recommended):**
```go
// when using http_client directly
resp, err := client.Post(ctx, "/upload",
    http_client.WithFormData(formData),
)
```

**Use `Build()` (for swagger/external clients):**
```go
// when using swagger-generated or other external clients
// that expect (contentType, body) parameters
body, contentType, err := formData.Build()
response, err := externalClient.UploadWithBody(ctx, contentType, body)
```

**Why does `Build()` exist?**

Swagger-generated clients (e.g. `oapi-codegen`) create methods like:
```go
UploadWithBodyWithResponse(ctx, params, contentType string, body io.Reader)
```

They require passing `contentType` and `body` separately instead of accepting a FormData object. The `Build()` method extracts these values for passing to such clients.
