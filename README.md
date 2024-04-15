[![build-img]][build-url]
[![pkg-img]][pkg-url]
[![reportcard-img]][reportcard-url]

# Client
Think of `httputil.Client` as a base or abstract client to be wrapped inside your real clients in place of `http.Client`.  For an example see example/client.go which can also work as a boilerplate to start your own client.

## Why?
I found myself copying large portions from previously implemented clients into new clients often, and so I figured it was time to make a package which would contain those basic features that could be used inside those future clients I will write rather than copying the code over and over again.

## Types
```go
package httputil

type (
	Client struct {
		ReqHeaders // stores headers and allows for client.AddHeader and client.SetHeader
		HttpClient   httpClient // *http.Client
		Host         string
		log          sLogger
		RateLimiter  *RateLimiter
		RetriesOn429 int
	}
	httpClient interface { // *http.Client
		Do(req *http.Request) (*http.Response, error)
	}
	LevelLogger interface { // slog.Logger
		Info(msg string, args ...any)
		Warn(msg string, args ...any)
		Error(msg string, args ...any)
	}
)
```

## Constructors and builder methods
```go
package httputil

func NewClient() Client
func (c Client) WithHost(string) Client
func (c Client) WithPathPrefix(v string) Client
func (c Client) WithLogger(LevelLogger) Client
func (c Client) WithHttpClient(httpClient) Client
func (c Client) WithRateLimiter(*RateLimiter) Client
func (c Client) With429Retry(int) Client
func (c Client) WithHeader(http.Header) Client
func (c Client) WithSetHeader(k string, v ...string) Client
func (c Client) Clone() Client
```

## Useful Methods
```go
package httputil

func (c Client) DoReq(ctx context.Context, method string, r Request, out, errRes any) (*http.Response, error)
func (c Client) Request(ctx context.Context, method, uri string, headers http.Header, body any) (*http.Request, error)
func (c Client) DoAndDecode(req *http.Request, out, errRes any) (*http.Response, error)
func (c Client) Do(req *http.Request) (*http.Response, error)
```

`Do()` Will append any headers set on the client to the request and uses the RateLimiter to allow for 429 retry and request throttling. It has the same signature as on `http.Client` so if you have client that takes a base client with that interface you could inject this as the base client just to get the RateLimiting feature alone.  It will also log all Request/Responses as Info if < 400, as Warn if < 500, else as error level, so be sure to set the logger level to control what you see.

`DoAndDecode()` calls `Do()` but then decodes the request body into `out` unless StatusCode >= 400 then into `errRes`.  It does this while leaving the response body so that it can still be read later if you wish.

`Request()` is just a simple request builder that will prepend the configured `Host` onto the uri for you along with adding the headers passed in and the headers stored on the client itself.

`DoReq()` is the method that saves you a lot of boilerplate in your client if you use it.  Just have your request models implement Request and this does much of the work for you.  It validates the model using `r.Validate()` then builds the request with `r.Path().WithHost(c.Host)` and `r.Header()`.  After the request is done it will decode into `out` or `errRes` depending on if the status is < 400 or not.  In a recent project this allowed me to implement an action method on my client this way where `c.client` is the `httputil.Client`

## Example

```go
package example

type RPM struct {
    client httputil.Client
}

func (c RPM) StoreProperty(ctx context.Context, r StorePropertyReq) (*StorePropertyRes, error) {
    var out StorePropertyRes
    if err := c.do(ctx, http.MethodPost, &r, &out); err != nil {
        return nil, err
    }
    return &out, nil
}

func (c RPM) do(ctx context.Context, method string, r httputil.Request, out any) error {
    var er ErrorResponse
    _, err := c.client.DoReq(ctx, method, r, &out, &er)
    if err != nil {
        return err
    }
    if err := er.ErrorOrNil(); err != nil {
        return err
    }
    return nil
}
```

Now that `do` method can be used for all of my action methods and all my requests are validated, constructed properly, and decoded properly.

# RateLimiter
The `RateLimiter` is a nil safe wrapper for `rate.Limiter`.  Both of which allow you to define a `limit float64` and `burst int` however it is important to note that `limit` and `burst` are not well named and may not mean exactly what you think they mean.  Rather think of it this way

- `burst` = bucket size: This is the maximum number of calls you can make when no wait at all.
- `limit` = bucket refill rate: This is how often tokens are released to the bucket

so suppose you had an endless loop making calls just as fast as it can.  Then `rate.NewLimiter(rate.Limit(10.0), 10)` has a limit (refill rate) of 10.0 and a burst (bucket size) of 10.  This would mean that 10 calls would fire instantly to use up all the tokens in the bucket, then the refil rate would release 10 tokens per second so they can be used again.  This results in 20 calls going out that first second even though you defined a `limit` of 10 per second.  However after that first burst with an endless loop waiting for tokens every second after that first second would only ever be able to do 10 calls because the tokens are always all used up and only released at a rate of 10 per second, so one every 0.1 seconds is released.

## Why not just use rate.Limit?
You can, `httputil{Limiter: rate.NewLimiter(...)}` works fine  The only advantage to this custom type that wraps it is that it is nilsafe and that you can configure a `changePercent` then use the `SlowDown` method when you encounter 429's to auto adapt to the limit imposed by whatever server you are calling.

## Constructors and options
```go
package httputil

func NewRateLimiter(limit float64, options ...RateLimitOption) *RateLimiter
func RateLimitChangePercent(percent float64) RateLimitOption
func RateLimitBurst(burst int) RateLimitOption
```

## Useful Methods
```go
package httputil

// Do simply calls `Wait` for you before executing the doFn
func (r *RateLimiter) Do(ctx context.Context, doFn func() error) error
func (r *RateLimiter) Wait(ctx context.Context) error
func (r *RateLimiter) SlowDown() // decrease the rate by ChangePercent
func (r *RateLimiter) SpeedUp()  // increase the rate by ChangePercent
```

# Path
The `Path` type is a URL builder allowing you to define a template with path arg placeholders, params for those path args, query args, baseURL (host) and prefix such as v1 or v2 etc.

This logic is moved here from https://github.com/tempcke/path and is especially helpful when having your request models implement `httputil.Request` and using the `DoReq` method on the `httputil.Client` to simplify your client building experience.

## Examples
```go
package httputil_test

func ExamplePath() {
	const pathFoo = "/foo/:foo"
	uri := httputil.NewPath(pathFoo).
		WithParam(":foo", "bar")
	fmt.Println(uri.String())
	// Output: /foo/bar
}
func ExamplePath_WithQuery() {
	const pathFooBarBaz = "/foo/:foo/bar/:bar/:baz"
	uri := httputil.NewPath(pathFooBarBaz).
		WithBaseURL("https://example.com").
		WithPrefix("v1").
		WithParam(":foo", "p1").
		WithParams(map[string]string{
			"bar": "p2",
			"baz": "p3",
		}).
		WithQuery("id", "1", "2").
		WithQuery("a", "A").
		WithQueryArgs(map[string]string{
			"b": "B",
			"c": "C",
		})
	fmt.Println(uri.String())
	// Output: https://example.com/v1/foo/p1/bar/p2/p3?a=A&b=B&c=C&id=1&id=2
}
```

## Constructors
```go
package httputil

func NewPath(template string) Path
```

## Constructor Methods
```go
package httputil

func (p Path) WithBaseURL(url string) Path
func (p Path) WithPrefix(basePath string) Path
func (p Path) WithParam(param, value string) Path
func (p Path) WithParams(params map[string]string) Path
func (p Path) WithQuery(key string, values ...string) Path
func (p Path) WithQueryArgs(args map[string]string) Path
func (p Path) WithQueryValues(query url.Values) Path
```

# Request

```go
package httputil

type (
	Request interface {
		Path()     Path
		Header()   http.Header // embed ReqHeaders into your model for this
		Validate() error
	}
	ReqHeaders struct {
		ReqID string // optional req header
		h     http.Header
	}
)
```
Your request model can represent everything needed to make the request.  Including path args, query args, headers, and finally the payload.  To exclude fields from the payload you entire make them private or add a - json tag `json:"-"`.

By having your request models implement `httputil.Request` interface it allows you to keep everything together.  Each request model can define the path, and headers used for the request.  You won't always want this of course, but I've found for me personally that I use it often.  Your request model may have some fields that go in the path, body, or even headers.  Using json tags you can define which are not for the body, and with Path() and Header() you can correctly build those.

Another advantage is that when building your client which uses `httputil.Client` under the hood, it can call `httputil.Client.DoReq()` which calls `Validate()` first, then builds the uri via `Path().WithBaseURL(c.Host).String()` and finally adds all the headers from `Header()` to the `http.Request` that it constructs.  This means you will not have to repeat all of these steps in each client action method and that they will behave the same for each.

## ReqHeaders

By embedding  `ReqHeaders` it makes it easy for a caller to add custom headers to their request if they desire to without having to have extra fields.  Also if you are not adding any headers from the struct fields you do not need to implement `Header()` as it is already implemented in `ReqHeaders` and will work fine when you pass a pointer of your request to `DoReq`

`ReqID` is a field in ReqHeaders.  So when you embed it then you can set `myreq.ReqID = uuid.NewString()` for example and it automatically adds it to the `myreq.Headers()` response using `X-Request-ID` as the key

## Example
```go
package api

type (
	Person struct {
		ID        string `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	StorePersonReq struct {
		httputil.ReqHeaders `json:"-"`
		Person Person       `json:"person"`
	}
)

var _ httputil.Request = (*StorePersonReq)(nil)

func (r StorePersonReq) Path() httputil.Path {
	return httputil.NewPath("/person/:personID").
		WithParam(":personID", r.Person.ID)
}
func (r StorePersonReq) Validate() error {
	if r.Person.ID == "" {
		return errors.New("missing ID")
	}
	if r.Person.FirstName == "" {
		return errors.New("missing first name")
	}
	if r.Person.LastName == "" {
		return errors.New("missing last name")
	}
	return nil
}
```

Notice I did not implement `Header()` because it is already in `ReqHeaders` however if there was a strict field that needed to be passed as a header I could implement Header(), have it add it like so

```go
package api

func (r StorePersonReq) Header() http.Header {
	r.ReqHeaders.Set("X-Force", r.shouldForce)
	return r.ReqHeaders.Header()
}
```


[build-img]: https://github.com/tempcke/httputil/actions/workflows/test.yml/badge.svg
[build-url]: https://github.com/tempcke/httputil/actions
[pkg-img]: https://pkg.go.dev/badge/tempcke/httputil
[pkg-url]: https://pkg.go.dev/github.com/tempcke/httputil
[reportcard-img]: https://goreportcard.com/badge/tempcke/httputil
[reportcard-url]: https://goreportcard.com/report/tempcke/httputil