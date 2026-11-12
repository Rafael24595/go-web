package router

import (
	stdlog "log"

	"context"
	"errors"
	"time"

	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rafael24595/go-collections/collection"
	"github.com/Rafael24595/go-log/log"
	"github.com/Rafael24595/go-log/log/record"
	"github.com/Rafael24595/go-web/router/configuration"
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

// OnListenFn receives the address a server is listening on; see Router.OnListen.
type OnListenFn func(net.Addr)

// ErrTLSStrategyRequired is returned by ListenTLS and ListenWithTLS when the
// TLS strategy is nil.
var ErrTLSStrategyRequired = errors.New("router: a TLS strategy is required")

// HandlerOptions represents advanced configuration for a route handler.
//
// It allows attaching optional handlers for:
//   - request context initialization
//   - error handling
//   - panic recovery
//
// Use NewHandlerOptions to create a new instance, and the builder-style
// methods (Context, Error, Panic) to configure it.
type HandlerOptions struct {
	handler RequestHandler
	context *contextHandler
	error   *errorHandler
	panic   *panicHandler
}

// NewHandlerOptions creates a new HandlerOptions instance for the given
// route handler. By default, no context, error, or panic handlers are set.
func NewHandlerOptions(handler RequestHandler) *HandlerOptions {
	return &HandlerOptions{
		handler: handler,
	}
}

// Context sets a context initializer for the handler.
//
// The contextualizer runs before the route logic and can build a request-
// scoped context dictionary (e.g., user info, DB connections, metadata).
//
// Returns the HandlerOptions itself for fluent configuration.
func (h *HandlerOptions) Context(context *contextHandler) *HandlerOptions {
	h.context = context
	return h
}

// Error sets a custom error handler for the route.
//
// The error handler runs when the route handler returns an error or fails.
//
// Returns the HandlerOptions itself for fluent configuration.
func (h *HandlerOptions) Error(error *errorHandler) *HandlerOptions {
	h.error = error
	return h
}

// Panic sets a custom panic handler for the route.
//
// The panic handler is called if the route panics during execution.
// This allows fine-grained recovery logic for individual routes.
//
// Returns the HandlerOptions itself for fluent configuration.
func (h *HandlerOptions) Panic(panic *panicHandler) *HandlerOptions {
	h.panic = panic
	return h
}

// Router is the main entry point for defining and running HTTP routes.
//
// It provides methods to register routes, contextualizers, error handlers,
type Router struct {
	logger log.Log

	server          *http.ServeMux
	ctx             context.Context
	shutdownTimeout time.Duration

	contextualizer       collection.IDictionary[string, contextHandler]
	groupContextualizers collection.IDictionary[string, collection.Vector[RequestHandler]]
	errors               collection.IDictionary[string, errorHandler]
	panics               collection.IDictionary[string, panicHandler]
	routes               collection.IDictionary[string, RequestHandler]

	basePath string
	cors     *Cors

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
		server:               server,
		ctx:                  context.Background(),
		shutdownTimeout:      DefaultShutdownTimeout,
		contextualizer:       collection.DictionaryEmpty[string, contextHandler](),
		groupContextualizers: collection.DictionaryEmpty[string, collection.Vector[RequestHandler]](),
		errors:               collection.DictionaryEmpty[string, errorHandler](),
		panics:               collection.DictionaryEmpty[string, panicHandler](),
		routes:               collection.DictionaryEmpty[string, RequestHandler](),
		basePath:             "",
		cors:                 EmptyCors(),
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
		result, _ := r.groupContextualizers.
			PutIfAbsent(v, *collection.VectorEmpty[RequestHandler]())
		result.Append(handler)
		r.groupContextualizers.Put(v, result)
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
		result, _ := r.groupContextualizers.
			PutIfAbsent(v, *collection.VectorEmpty[RequestHandler]())
		result.Append(handler)
		path := fmt.Sprintf("%s%s", r.basePath, v)
		r.groupContextualizers.Put(path, result)
		r.docViewer.RegisterGroup(path, doc)
	}
	return r
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

// Route registers a basic route with default handler options.
//
// This is a shorthand for RouteWithOptions where only the handler is provided.
// Use this when you don’t need documentation or advanced configuration.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Route(method string, handler RequestHandler, pattern string, params ...any) *Router {
	return r.RouteWithOptions(method, NewHandlerOptions(handler), pattern, params...)
}

// RouteWithOptions registers a basic route with advanced handler options.
//
// Use this when you need to configure context, error, or panic handling
// at the route level, without attaching documentation.
//
// Returns the Router itself for fluent configuration.
func (r *Router) RouteWithOptions(method string, options *HandlerOptions, pattern string, params ...any) *Router {
	return r.route(method, pattern, options, docs.DocOperation{}, params...)
}

// RouteDocument registers a documented route with default handler options.
//
// This is a shorthand for RouteDocumentWithOptions where only the handler
// is provided. Use this when you don’t need custom context, error, or panic
// handlers.
//
// Returns the Router itself for fluent configuration.
func (r *Router) RouteDocument(method string, handler RequestHandler, pattern string, doc docs.DocRoute) *Router {
	return r.RouteDocumentWithOptions(method, NewHandlerOptions(handler), pattern, doc)
}

// RouteDocumentWithOptions registers a documented route with advanced handler options.
//
// The documentation payload describes request parameters, queries, cookies,
// file uploads, request body, and possible responses. HandlerOptions may
// define custom context, error, or panic behavior for this route.
//
// Returns the Router itself for fluent configuration.
func (r *Router) RouteDocumentWithOptions(method string, options *HandlerOptions, pattern string, doc docs.DocRoute) *Router {
	params := make([]any, 0)
	for _, p := range doc.Parameters {
		params = append(params, p.Code)
	}

	docRoute := docs.DocOperation{
		Description: doc.Description,
		Parameters:  doc.Parameters,
		Query:       doc.Query,
		Cookies:     doc.Cookies,
		Files:       doc.Files,
		Request:     doc.Request,
		Responses:   doc.Responses,
		Tags:        doc.Tags,
	}

	return r.route(method, pattern, options, docRoute, params...)
}

func (r *Router) route(method string, pattern string, options *HandlerOptions, doc docs.DocOperation, params ...any) *Router {
	route := r.patternKey(method, pattern, params...)

	if options != nil && options.context != nil {
		r.contextualizer.Put(route, *options.context)
	}

	if options != nil && options.error != nil {
		r.errors.Put(route, *options.error)
	}

	if options != nil && options.panic != nil {
		r.panics.Put(route, *options.panic)
	}

	r.routes.Put(route, options.handler)
	r.server.HandleFunc(route, r.handler)

	doc.Method = method
	doc.BasePath = r.basePath
	doc.Path = fmt.Sprintf(pattern, params...)

	r.docViewer.RegisterRoute(doc)

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

// Handler returns the router as an http.Handler with its middlewares (CORS)
// applied. Prefer Listen, ListenTLS and ListenWithTLS, which also manage the
// server's lifetime; use Handler for tests (httptest.NewServer) or to mount
// the router inside a server you build yourself.
func (r *Router) Handler() http.Handler {
	middleware := r.middlewares()
	return applyMiddleware(r.server, middleware)
}

// Listen starts an HTTP server on addr (":8080", "127.0.0.1:8080"; a number
// alone is a port) and blocks until it stops. It returns nil when the router's
// context ends the server, and an error if addr cannot be used, the server
// fails, or in-flight requests outlive the shutdown timeout.
//
//	router.Listen(":8080", router.WithReadHeaderTimeout(10*time.Second))
func (r *Router) Listen(addr string, options ...ServerOption) error {
	addr = normalizeAddr(addr)
	middleware := r.middlewares()

	server := r.newServer(addr, middleware, options...)
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
	middleware := r.middlewares()

	server := r.newServer(addr, middleware, options...)

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

	redirect := []middleware{
		httpsRedirectMiddleware(addrTLS),
	}

	secure := r.middlewares()

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	strategy, err := tls()
	if err != nil {
		return err
	}

	plain := make(chan error, 1)
	go func() {
		defer cancel()
		server := r.newServer(addr, redirect, options...)
		strategy := servePlain()
		plain <- r.run(ctx, server, strategy)
	}()

	server := r.newServer(addrTLS, secure, options...)

	err = r.run(ctx, server, strategy)
	cancel()

	return errors.Join(err, <-plain)
}

func (r *Router) middlewares() []middleware {
	if r.cors.IsEmpty() {
		return nil
	}

	return []middleware{
		corsMiddleware(*r.cors),
	}
}

func (r *Router) newServer(
	addr string,
	middleware []middleware,
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
	server.Handler = applyMiddleware(r.server, middleware)
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
		<-stopped
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

	config := configuration.Instance()
	if config.Dev() && config.TraceRequest() {
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
