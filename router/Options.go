package router

import (
	"fmt"

	"github.com/Rafael24595/go-web/router/docs"
)

type routeSpec struct {
	method  string
	pattern string
	params  []any
}

type resultSpec struct {
	route string
	routeSpec
}

type RouteOption struct {
	resolve  func(spec *routeSpec)
	register func(r *Router, spec resultSpec)
}

func WithParams(params ...any) RouteOption {
	return RouteOption{
		resolve: func(register *routeSpec) {
			register.params = append(register.params, params...)
		},
	}
}

func WithContext(handler contextHandler) RouteOption {
	if handler == nil {
		return RouteOption{}
	}

	return RouteOption{
		register: func(r *Router, spec resultSpec) {
			r.contextualizer.Put(spec.route, handler)
		},
	}
}

func WithError(handler errorHandler) RouteOption {
	if handler == nil {
		return RouteOption{}
	}

	return RouteOption{
		register: func(r *Router, spec resultSpec) {
			r.errors.Put(spec.route, handler)
		},
	}
}

func WithPanic(handler panicHandler) RouteOption {
	if handler == nil {
		return RouteOption{}
	}

	return RouteOption{
		register: func(r *Router, spec resultSpec) {
			r.panics.Put(spec.route, handler)
		},
	}
}

func WithDocs(doc docs.DocRoute) RouteOption {
	return RouteOption{
		resolve: func(register *routeSpec) {
			for _, p := range doc.Parameters {
				register.params = append(register.params, p.Code)
			}
		},
		register: func(r *Router, spec resultSpec) {
			path := fmt.Sprintf(spec.pattern, spec.params...)
			r.docViewer.RegisterRoute(docs.DocOperation{
				Description: doc.Description,
				Method:      spec.method,
				BasePath:    r.basePath,
				Path:        path,
				Parameters:  doc.Parameters,
				Query:       doc.Query,
				Cookies:     doc.Cookies,
				Files:       doc.Files,
				Request:     doc.Request,
				Responses:   doc.Responses,
				Tags:        doc.Tags,
			})
		},
	}
}
