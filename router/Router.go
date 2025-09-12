package router

import (
	"fmt"
	stdlog "log"
	"net/http"
	"strings"

	"github.com/Rafael24595/go-collections/collection"
	"github.com/Rafael24595/go-web/router/docs"
	"github.com/Rafael24595/go-web/router/log"
	"github.com/Rafael24595/go-web/router/result"
)

type Context = collection.IDictionary[string, any]
type contextHandler = func(http.ResponseWriter, *http.Request) (Context, error)
type requestHandler = func(http.ResponseWriter, *http.Request, Context) result.Result
type errorHandler = func(http.ResponseWriter, *http.Request, Context, result.Result)
type panicHandler = func(http.ResponseWriter, *http.Request, any)

const BASE = "$BASE"

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
	handler requestHandler
	context *contextHandler
	error   *errorHandler
	panic   *panicHandler
}

// NewHandlerOptions creates a new HandlerOptions instance for the given
// route handler. By default, no context, error, or panic handlers are set.
func NewHandlerOptions(handler requestHandler) *HandlerOptions {
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

type Router struct {
	logger               log.Log
	contextualizer       collection.IDictionary[string, contextHandler]
	groupContextualizers collection.IDictionary[string, collection.Vector[requestHandler]]
	errors               collection.IDictionary[string, errorHandler]
	panics               collection.IDictionary[string, panicHandler]
	routes               collection.IDictionary[string, requestHandler]
	basePath             string
	cors                 *Cors
	docViewer            docs.IDocViewer
}

// NewRouter creates and initializes a new Router instance with sensible defaults.
//
// The router comes with:
//   - a default logger
//   - empty collections for routes, error handlers, panic handlers, and contextualizers
//   - an empty base path
//   - default CORS configuration
//   - a "no-op" documentation viewer
//
// Use this function as the entry point to build and configure a new Router.
func NewRouter() *Router {
	return &Router{
		logger:               log.DefaultLogger(),
		contextualizer:       collection.DictionaryEmpty[string, contextHandler](),
		groupContextualizers: collection.DictionaryEmpty[string, collection.Vector[requestHandler]](),
		errors:               collection.DictionaryEmpty[string, errorHandler](),
		routes:               collection.DictionaryEmpty[string, requestHandler](),
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
		http.HandleFunc(pattern, v.Handler)
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
	http.Handle(fmt.Sprintf("GET %s", route), http.StripPrefix(route, fs))
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
	r.contextualizer.Put(BASE, handler)
	return r
}

// GroupContextualizer registers a group-specific contextualizer
// without documentation.
//
// A group contextualizer executes before all routes belonging to the
// specified group.
//
// Returns the Router itself for fluent configuration.
func (r *Router) GroupContextualizer(handler requestHandler, group ...string) *Router {
	for _, v := range group {
		result, _ := r.groupContextualizers.
			PutIfAbsent(v, *collection.VectorEmpty[requestHandler]())
		result.Append(handler)
		r.groupContextualizers.Put(v, *result)
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
func (r *Router) GroupContextualizerDocument(handler requestHandler, doc docs.DocGroup, group ...string) *Router {
	for _, v := range group {
		result, _ := r.groupContextualizers.
			PutIfAbsent(v, *collection.VectorEmpty[requestHandler]())
		result.Append(handler)
		path := fmt.Sprintf("%s%s", r.basePath, v)
		r.groupContextualizers.Put(path, *result)
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
	r.errors.Put(BASE, handler)
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
	r.panics.Put(BASE, handler)
	return r
}

// Route registers a basic route with default handler options.
//
// This is a shorthand for RouteWithOptions where only the handler is provided.
// Use this when you don’t need documentation or advanced configuration.
//
// Returns the Router itself for fluent configuration.
func (r *Router) Route(method string, handler requestHandler, pattern string, params ...any) *Router {
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
func (r *Router) RouteDocument(method string, handler requestHandler, pattern string, doc docs.DocRoute) *Router {
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
	for p := range doc.Parameters {
		params = append(params, p)
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
	http.HandleFunc(route, r.handler)

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

// Listen starts an HTTP server on the given host.
//
// Example:
//
//	router.Listen(":8080")
//
// CORS and other startup middlewares are automatically applied.
func (r *Router) Listen(host string) error {
	middleware := []middleware{
		corsMiddleware(r.cors),
	}
	return r.listen(host, middleware)
}

// ListenTLS starts an HTTPS server on the given host with TLS enabled.
//
// Requires a certificate and private key.
//
// Example:
//
//	router.ListenTLS(":8443", "server.crt", "server.key")
func (r *Router) ListenTLS(hostTLS, certTLS, keyTLS string) error {
	middleware := []middleware{
		corsMiddleware(r.cors),
	}
	return r.listenTLS(hostTLS, certTLS, keyTLS, middleware)
}

// ListenWithTLS starts both HTTP and HTTPS servers in parallel.
//
// The HTTP server listens on the first host and automatically redirects
// all requests to the HTTPS server. The HTTPS server listens on the
// second host using the provided certificate and key.
//
// Example:
//
//	router.ListenWithTLS(":8080", ":8443", "server.crt", "server.key")
func (r *Router) ListenWithTLS(host, hostTLS, certTLS, keyTLS string) error {
	middleware := []middleware{
		corsMiddleware(r.cors),
		httpsRedirectMiddleware(hostTLS),
	}

	go func() {
		if err := r.listen(host, middleware); err != nil {
			r.logger.Error(err)
		}
	}()

	return r.ListenTLS(hostTLS, certTLS, keyTLS)
}

func (r *Router) listenTLS(hostTLS, certTLS, keyTLS string, middleware []middleware) error {
	server := &http.Server{
		Addr:     hostTLS,
		Handler:  applyMiddleware(http.DefaultServeMux, middleware),
		ErrorLog: stdlog.New(r.logger, "", 0),
	}

	r.logger.Messagef("The app is listen at: %s with TLS", hostTLS)
	return server.ListenAndServeTLS(certTLS, keyTLS)
}

func (r *Router) listen(host string, middleware []middleware) error {
	server := &http.Server{
		Addr:     host,
		Handler:  applyMiddleware(http.DefaultServeMux, middleware),
		ErrorLog: stdlog.New(r.logger, "", 0),
	}

	r.logger.Messagef("The app is listen at: %s", host)
	return server.ListenAndServe()
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

	handler, ok := r.routes.Get(req.Pattern)
	if !ok {
		r.logger.Errors("Request handler not found")
	}

	ctx, ctxResult := r.initializeContext(wrt, req)
	if ctxResult != nil {
		r.manageErr(wrt, req, ctx, *ctxResult)
		return
	}

	result := (*handler)(wrt, req, ctx)
	if result.Ok() {
		r.manageOk(wrt, result)
		return
	}

	r.manageErr(wrt, req, ctx, result)
}

func (r *Router) initializeContext(wrt http.ResponseWriter, req *http.Request) (Context, *result.Result) {
	contextualizer, ok := r.contextualizer.Get(req.Pattern)
	if !ok {
		contextualizer, ok = r.contextualizer.Get(BASE)
	}

	var context Context
	context = collection.DictionaryEmpty[string, any]()
	if ok {
		var err error
		context, err = (*contextualizer)(wrt, req)
		if err != nil {
			r.logger.Error(err)
		}
	}

	group := strings.Split(req.Pattern, " ")[1]
	keys := r.groupContextualizers.KeysVector().Filter(func(key string) bool {
		return strings.HasPrefix(group, key)
	})

	for _, key := range keys.Collect() {
		funcs, ok := r.groupContextualizers.Get(key)
		if !ok {
			return context, nil
		}

		for _, f := range funcs.Collect() {
			result := f(wrt, req, context)

			if result.Err() {
				return context, &result
			}
		}
	}

	return context, nil
}

func (r *Router) manageOk(wrt http.ResponseWriter, result result.Result) {
	encoder := result.Encoder()
	encode, err := encoder.Encode(result.Payload())
	if err != nil {
		http.Error(wrt, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.Err() {
		http.Error(wrt, string(encode), result.Status())
		return
	}

	wrt.WriteHeader(result.Status())

	for k, v := range encoder.Headers() {
		wrt.Header().Set(k, v)
	}

	_, err = wrt.Write(encode)
	if err != nil {
		r.logger.Errorf("Error writing response: %s", err.Error())
		return
	}
}

func (r *Router) manageErr(wrt http.ResponseWriter, req *http.Request, context Context, result result.Result) {
	errorHandler, ok := r.errors.Get(req.Pattern)
	if !ok {
		errorHandler, ok = r.errors.Get(BASE)
	}

	if ok {
		(*errorHandler)(wrt, req, context, result)
		return
	}

	r.manageOk(wrt, result)
}

func (r *Router) managePanic(wrt http.ResponseWriter, req *http.Request, rec any) {
	panicHandler, ok := r.panics.Get(req.Pattern)
	if !ok {
		panicHandler, ok = r.panics.Get(BASE)
	}

	if ok {
		(*panicHandler)(wrt, req, rec)
		return
	}

	message := fmt.Sprintf("Uncontrolled panic during resolution of '%s'", req.Pattern)
	http.Error(wrt, message, http.StatusInternalServerError)
}

func (r Router) patternKey(method, pattern string, params ...any) string {
	return fmt.Sprintf("%s %s%s", method, r.basePath, fmt.Sprintf(pattern, params...))
}
