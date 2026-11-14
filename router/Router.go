package router

import (
	stdlog "log"

	"context"
	"errors"
	"time"

	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Rafael24595/go-assert/invariant"
	"github.com/Rafael24595/go-collections/collection"
	"github.com/Rafael24595/go-log/log"
	"github.com/Rafael24595/go-log/log/record"
	"github.com/Rafael24595/go-web/router/docs"
	"github.com/Rafael24595/go-web/router/result"
)

const (
	// Base is the key used to register global handlers (contextualizer, error, panic)
	Base = "$BASE"
	// DefaultShutdownTimeout is how long in-flight requests get to finish when
	// the router's context is done, unless ShutdownTimeout sets another value.
	DefaultShutdownTimeout = 5 * time.Second
)

type contextHandler = func(http.ResponseWriter, *http.Request) (*Context, error)
type RequestHandler = func(http.ResponseWriter, *http.Request, *Context) result.Result
type errorHandler = func(http.ResponseWriter, *http.Request, *Context, result.Result)
type panicHandler = func(http.ResponseWriter, *http.Request, any)

// Middleware wraps the router's handler. Registered with Router.Use.
type Middleware func(http.Handler) http.Handler

// OnListenFn receives the address a server is listening on; see Router.OnListen.
type OnListenFn func(net.Addr)

// ErrTLSStrategyRequired is returned by ListenTLS and ListenWithTLS when the
// TLS strategy is nil.
var ErrTLSStrategyRequired = errors.New("router: a TLS strategy is required")

// Router is the main entry point for defining and running HTTP routes.
//
// It provides methods to register routes, contextualizers, error handlers,
type Router struct {
	logger log.Log
	trace  bool

	server          *http.ServeMux
	ctx             context.Context
	shutdownTimeout time.Duration

	contextualizer       collection.IDictionary[string, contextHandler]
	groupContextualizers collection.IDictionary[string, collection.Vector[RequestHandler]]
	errors               collection.IDictionary[string, errorHandler]
	panics               collection.IDictionary[string, panicHandler]
	routes               collection.IDictionary[string, RequestHandler]

	basePath   string
	middleware []Middleware
	cors       *Cors

	docViewer docs.IDocViewer
	onListen  func(net.Addr)
}

// NewRouter creates a Router that registers its routes on http.DefaultServeMux.
// Only one such router can declare a given route per process; use
// NewRouterWithMux when several routers live in the same process, as in tests.
//
// The router comes with:
//   - a default logger
//   - an HTTP server using the default ServeMux
//   - empty collections for routes, error handlers, panic handlers, and contextualizers
//   - an empty base path
//   - default CORS configuration
//   - a "no-op" documentation viewer
//
// Use this function as the entry point to build and configure a new Router.
func NewRouter() *Router {
	return NewRouterWithMux(http.DefaultServeMux)
}

// NewRouterWithMux creates a Router that registers its routes on mux.
// A nil mux creates a new one.
func NewRouterWithMux(server *http.ServeMux) *Router {
	if server == nil {
		server = http.NewServeMux()
	}

	return &Router{
		logger:               log.Default(),
		trace:                false,
		server:               server,
		ctx:                  context.Background(),
		shutdownTimeout:      DefaultShutdownTimeout,
		contextualizer:       collection.DictionaryEmpty[string, contextHandler](),
		groupContextualizers: collection.DictionaryEmpty[string, collection.Vector[RequestHandler]](),
		errors:               collection.DictionaryEmpty[string, errorHandler](),
		panics:               collection.DictionaryEmpty[string, panicHandler](),
		routes:               collection.DictionaryEmpty[string, RequestHandler](),
		basePath:             "",
		cors:                 NewCors(),
		docViewer:            docs.VoidViewer(),
	}
}

// Logger sets the logger used by the Router for internal messages,
// error reporting, and panic recovery.
//
// By default, a simple logger is provided. This method allows injecting
// a custom logger implementation.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Logger(logger log.Log) *Router {
	r.logger = logger
	return r
}

// TraceRequests logs every request the router handles, with its method,
// path and duration. It is off by default.
//
// Returns the Router itself for fluent configuration.
func (r *Router) TraceRequests(trace bool) *Router {
	r.trace = trace
	return r
}

// Context ties the servers started by Listen, ListenTLS and ListenWithTLS
// to ctx: when ctx is done they stop accepting connections, give in-flight
// requests up to the shutdown timeout to finish, and the Listen call returns.
// A context that is already done makes Listen return at once. Without a
// context the servers run until the process ends.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Context(ctx context.Context) *Router {
	if ctx == nil {
		ctx = context.Background()
	}

	r.ctx = ctx
	return r
}

// ShutdownTimeout sets how long in-flight requests get to finish once the
// router's context is done. The default is DefaultShutdownTimeout.
//
// Returns the Router itself for fluent configuration.
func (r *Router) ShutdownTimeout(timeout time.Duration) *Router {
	r.shutdownTimeout = timeout
	return r
}

// OnListen registers a function called with the actual address once a server
// is listening and before it serves requests: useful to open a browser, to
// learn the port chosen for ":0", or to signal readiness. It is not called
// when the address cannot be used. ListenWithTLS calls it once per server.
//
// Returns the Router itself for fluent configuration.
func (r *Router) OnListen(fn OnListenFn) *Router {
	r.onListen = fn
	return r
}

// DocViewer registers a documentation viewer responsible for exposing
// documentation endpoints.
//
// When set, the viewer’s handlers are mounted in the underlying HTTP mux,
// and new routes will automatically register themselves in the viewer.
//
// Returns the Router itself for fluent configuration.
func (r *Router) DocViewer(viewer docs.IDocViewer) *Router {
	for _, v := range viewer.Handlers() {
		pattern := fmt.Sprintf("%s %s", v.Method, v.Route)
		r.server.HandleFunc(pattern, v.Handler)
	}
	r.docViewer = viewer
	return r
}

// BasePath sets a common path prefix for all registered routes.
//
// For example, if the base path is "/api/v1", then a registered route
// "/users" will be exposed as "/api/v1/users".
//
// Returns the Router itself for fluent configuration.
func (r *Router) BasePath(basePath string) *Router {
	r.basePath = basePath
	return r
}

// ResourcesPath serves static resources from the specified directory path.
//
// This is useful for exposing assets such as images, stylesheets, or
// JavaScript files. The resources are served under "/<path>/".
//
// Returns the Router itself for fluent configuration.
func (r *Router) ResourcesPath(path string) *Router {
	fs := http.FileServer(http.Dir(path))
	route := fmt.Sprintf("/%s/", path)
	r.server.Handle(fmt.Sprintf("GET %s", route), http.StripPrefix(route, fs))
	return r
}

// Contextualizer registers a global context initializer for all routes.
//
// A contextualizer is a function that builds a request-scoped context
// dictionary before the route handler executes. This can be used to
// inject dependencies, user session data, or request metadata.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Contextualizer(handler contextHandler) *Router {
	r.contextualizer.Put(Base, handler)
	return r
}

// GroupContextualizer registers a group-specific contextualizer
// without documentation.
//
// A group contextualizer executes before all routes belonging to the
// specified group.
//
// Returns the Router itself for fluent configuration.
func (r *Router) GroupContextualizer(handler RequestHandler, group ...string) *Router {
	for _, v := range group {
		r.addGroupContextualizer(r.basePath+v, handler)
	}
	return r
}

// GroupContextualizerDocument registers a group-specific contextualizer
// along with its documentation.
//
// This is useful for applying middleware-like logic to entire groups
// of routes (e.g., authentication, logging, or metrics) while also
// documenting the group in the API viewer.
//
// Returns the Router itself for fluent configuration.
func (r *Router) GroupContextualizerDocument(handler RequestHandler, doc docs.DocGroup, group ...string) *Router {
	for _, v := range group {
		path := r.basePath + v
		r.addGroupContextualizer(path, handler)
		r.docViewer.RegisterGroup(path, doc)
	}
	return r
}

func (r *Router) addGroupContextualizer(path string, handler RequestHandler) {
	handlers, _ := r.groupContextualizers.
		PutIfAbsent(path, *collection.VectorEmpty[RequestHandler]())

	handlers.Append(handler)
	r.groupContextualizers.Put(path, handlers)
}

// ErrorHandler registers a global error handler.
//
// The error handler is invoked when a route handler returns an error
// or when a contextualizer fails. If no handler is registered, errors
// are returned as plain HTTP responses.
//
// Returns the Router itself for fluent configuration.
func (r *Router) ErrorHandler(handler errorHandler) *Router {
	r.errors.Put(Base, handler)
	return r
}

// PanicHandler registers a global panic handler.
//
// The panic handler is invoked when a route handler or contextualizer panics
// during request processing.
//
// By default, the Router only logs panics and continues execution.
// Registering a panic handler lets you override this behavior.
//
// Returns the Router itself for fluent configuration.
func (r *Router) PanicHandler(handler panicHandler) *Router {
	r.panics.Put(Base, handler)
	return r
}

// Route registers a route; params fill the %s verbs of the pattern. The route
// is served but left out of the documentation viewer: use RouteDocument, or
// RouteWithOptions with WithDocs, to document it.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Route(method string, handler RequestHandler, pattern string, params ...any) *Router {
	return r.RouteWithOptions(method, handler, pattern, WithParams(params...))
}

// RouteDocument registers a route and documents it; it is a shorthand for
// RouteWithOptions with WithDocs.
//
// Returns the Router itself for fluent configuration.
func (r *Router) RouteDocument(method string, handler RequestHandler, pattern string, doc docs.DocRoute) *Router {
	return r.RouteWithOptions(method, handler, pattern, WithDocs(doc))
}

// RouteWithOptions registers a route configured by options: its parameters
// (WithParams), its own contextualizer, error or panic handler (WithContext,
// WithError, WithPanic) and its documentation (WithDocs).
//
// A nil handler is a programming error: debug builds panic, release builds
// skip the route.
//
// Returns the Router itself for fluent configuration.
func (r *Router) RouteWithOptions(method string, handler RequestHandler, pattern string, options ...RouteOption) *Router {
	if handler == nil {
		invariant.Unreachable(
			fmt.Sprintf("router: nil handler for %s %s", method, pattern),
		)
		return r
	}

	routeSpec := routeSpec{
		method:  method,
		pattern: pattern,
	}

	for _, option := range options {
		if option.resolve != nil {
			option.resolve(&routeSpec)
		}
	}

	route := r.patternKey(routeSpec.method, routeSpec.pattern, routeSpec.params...)

	r.routes.Put(route, handler)
	r.server.HandleFunc(route, r.handler)

	resultSpec := resultSpec{
		route:     route,
		routeSpec: routeSpec,
	}

	for _, option := range options {
		if option.register != nil {
			option.register(r, resultSpec)
		}
	}

	return r
}

// Use adds middlewares around every route served by Listen, ListenTLS,
// ListenWithTLS and Handler. The first one added is the outermost. They run
// after the built-in CORS handling: preflight requests are answered before
// reaching them, and their responses (a 403 from a Host check, for instance)
// already carry the CORS headers, so cross-origin clients can read them.
// They are not applied to the HTTP-to-HTTPS redirect server of ListenWithTLS.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Use(middlewares ...Middleware) *Router {
	for _, m := range middlewares {
		if m != nil {
			r.middleware = append(r.middleware, m)
		}
	}
	return r
}

// Cors configures the Router's CORS policy.
//
// This controls which origins, methods, headers, and credentials are
// allowed when handling cross-origin requests.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Cors(cors *Cors) *Router {
	r.cors = cors
	return r
}

// Handler returns the router as an http.Handler with CORS and the middlewares 
// added with Use applied. Prefer Listen, ListenTLS and ListenWithTLS, which 
// also manage the server's lifetime; use Handler for tests (httptest.NewServer)
// or to mount the router inside a server you build yourself.
func (r *Router) Handler() http.Handler {
	return r.serving()
}

// Listen starts an HTTP server on addr (":8080", "127.0.0.1:8080"; a number
// alone is a port) and blocks until it stops. It returns nil when the router's
// context ends the server, and an error if addr cannot be used, the server
// fails, or in-flight requests outlive the shutdown timeout.
//
//	router.Listen(":8080", router.WithReadHeaderTimeout(10*time.Second))
func (r *Router) Listen(addr string, options ...ServerOption) error {
	addr = normalizeAddr(addr)
	handler := r.serving()

	server := r.newServer(addr, handler, options...)
	strategy := servePlain()

	return r.run(r.ctx, server, strategy)
}

// ListenTLS is Listen over HTTPS, with the certificate given by tls.
//
//	router.ListenTLS(":8443", router.TLSFromFiles("server.crt", "server.key"))
func (r *Router) ListenTLS(
	addr string,
	tls TLSStrategy,
	options ...ServerOption,
) error {
	if tls == nil {
		return ErrTLSStrategyRequired
	}

	addr = normalizeAddr(addr)
	handler := r.serving()

	server := r.newServer(addr, handler, options...)

	strategy, err := tls()
	if err != nil {
		return err
	}

	return r.run(r.ctx, server, strategy)
}

// ListenWithTLS runs an HTTP server on addr that redirects every request
// (308) to an HTTPS server on addrTLS. When either server stops, the other
// one is stopped too.
//
//	router.ListenWithTLS(":8080", ":8443", router.TLSFromFiles("server.crt", "server.key"))
func (r *Router) ListenWithTLS(
	addr, addrTLS string,
	tls TLSStrategy,
	options ...ServerOption,
) error {
	if tls == nil {
		return ErrTLSStrategyRequired
	}

	addr = normalizeAddr(addr)
	addrTLS = normalizeAddr(addrTLS)

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	strategy, err := tls()
	if err != nil {
		return err
	}

	plain := make(chan error, 1)
	go func() {
		defer cancel()
		handler := applyInterceptor(
			r.server, httpsRedirectInterceptor(addrTLS),
		)

		server := r.newServer(addr, handler, options...)
		strategy := servePlain()

		plain <- r.run(ctx, server, strategy)
	}()

	handler := r.serving()
	server := r.newServer(addrTLS, handler, options...)

	err = r.run(ctx, server, strategy)
	cancel()

	return errors.Join(err, <-plain)
}

func (r *Router) serving() http.Handler {
	var handler http.Handler = r.server
	for _, v := range slices.Backward(r.middleware) {
		handler = v(handler)
	}

	if r.cors.IsEmpty() {
		return handler
	}

	return applyInterceptor(
		handler, corsInterceptor(*r.cors),
	)
}

func (r *Router) newServer(
	addr string,
	handler http.Handler,
	options ...ServerOption,
) *http.Server {
	server := &http.Server{}

	for _, option := range options {
		if option != nil {
			option(server)
		}
	}

	writer := log.NewWriterFromCategory(r.logger, record.WARNING)

	server.Addr = addr
	server.Handler = handler
	server.ErrorLog = stdlog.New(writer, "", 0)

	return server
}

func (r *Router) run(ctx context.Context, server *http.Server, strategy serveStrategy) error {
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	r.logger.Messagef("The app is listening at %s://%s", strategy.scheme, listener.Addr())

	if r.onListen != nil {
		r.onListen(listener.Addr())
	}

	done := make(chan struct{})
	stopped := make(chan error, 1)

	go func() {
		select {
		case <-ctx.Done():
		case <-done:
			stopped <- nil
			return
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout)
		defer cancel()
		stopped <- server.Shutdown(shutdownCtx)
	}()

	err = strategy.serve(server, listener)

	if !errors.Is(err, http.ErrServerClosed) {
		close(done)

		stopErr := <-stopped
		if stopErr != nil {
			return errors.Join(err, stopErr)
		}

		return err
	}

	return <-stopped
}

func normalizeAddr(addr string) string {
	if addr == "" {
		return addr
	}

	if _, err := strconv.Atoi(addr); err == nil {
		return net.JoinHostPort("", addr)
	}

	return addr
}

// ViewerSources retrieves the list of documentation sources currently
// available in the configured documentation viewer.
//
// Each entry contains metadata such as name, route, and description.
func (r *Router) ViewerSources() []docs.DocViewerSources {
	if r.docViewer == nil {
		return make([]docs.DocViewerSources, 0)
	}

	handlers := r.docViewer.Handlers()
	sources := make([]docs.DocViewerSources, len(handlers))
	for i, v := range handlers {
		sources[i] = docs.DocViewerSources{
			Name:        v.Name,
			Route:       v.Route,
			Description: v.Description,
		}
	}
	return sources
}

func (r *Router) handler(wrt http.ResponseWriter, req *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			r.managePanic(wrt, req, rec)
		}
	}()

	if r.trace {
		message := fmt.Sprintf("%s - %s", req.RemoteAddr, req.Pattern)
		r.logger.Custom("DEV-REQUEST", message)
	}

	handler, ok := r.routes.Get(req.Pattern)
	if !ok {
		http.NotFound(wrt, req)
		r.logger.Errors("Request handler not found")
		return
	}

	ctx, ctxResult := r.initializeContext(wrt, req)
	if ctxResult != nil {
		r.manageErr(wrt, req, ctx, *ctxResult)
		return
	}

	result := handler(wrt, req, ctx)
	if result.Ignore() {
		return
	}

	if result.Ok() {
		r.manageOk(wrt, req, result)
		return
	}

	r.manageErr(wrt, req, ctx, result)
}

func (r *Router) initializeContext(wrt http.ResponseWriter, req *http.Request) (*Context, *result.Result) {
	contextualizer, ok := r.contextualizer.Get(req.Pattern)
	if !ok {
		contextualizer, ok = r.contextualizer.Get(Base)
	}

	var ctx *Context
	if ok {
		var err error
		ctx, err = contextualizer(wrt, req)
		if err != nil {
			r.logger.Error(err)
		}
	}

	if ctx == nil {
		ctx = NewContext()
	}

	group := strings.Split(req.Pattern, " ")[1]
	keys := r.groupContextualizers.KeysVector().Filter(func(key string) bool {
		return strings.HasPrefix(group, key)
	})

	for _, key := range keys.Collect() {
		funcs, ok := r.groupContextualizers.Get(key)
		if !ok {
			return ctx, nil
		}

		for _, f := range funcs.Collect() {
			result := f(wrt, req, ctx)

			if result.Err() {
				return ctx, &result
			}
		}
	}

	return ctx, nil
}

func (r *Router) manageOk(wrt http.ResponseWriter, req *http.Request, result result.Result) {
	encoder := result.Encoder()
	encode, err := encoder.Encode(result.Payload())
	if err != nil {
		http.Error(wrt, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.Err() {
		wrt.Header().Set("X-Content-Type-Options", "nosniff")
	}

	if result.File() {
		http.ServeFile(wrt, req, string(encode))
		return
	}

	for k, v := range encoder.Headers() {
		wrt.Header().Set(k, v)
	}

	wrt.WriteHeader(result.Status())

	_, err = wrt.Write(encode)
	if err != nil {
		r.logger.Errorf("Error writing response: %s", err.Error())
		return
	}
}

func (r *Router) manageErr(wrt http.ResponseWriter, req *http.Request, context *Context, result result.Result) {
	errorHandler, ok := r.errors.Get(req.Pattern)
	if !ok {
		errorHandler, ok = r.errors.Get(Base)
	}

	if ok {
		errorHandler(wrt, req, context, result)
		return
	}

	r.manageOk(wrt, req, result)
}

func (r *Router) managePanic(wrt http.ResponseWriter, req *http.Request, rec any) {
	panicHandler, ok := r.panics.Get(req.Pattern)
	if !ok {
		panicHandler, ok = r.panics.Get(Base)
	}

	if ok {
		panicHandler(wrt, req, rec)
		return
	}

	message := fmt.Sprintf("Uncontrolled panic during resolution of '%s'", req.Pattern)
	http.Error(wrt, message, http.StatusInternalServerError)
}

func (r Router) patternKey(method, pattern string, params ...any) string {
	return fmt.Sprintf("%s %s%s", method, r.basePath, fmt.Sprintf(pattern, params...))
}
