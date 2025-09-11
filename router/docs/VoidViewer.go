package docs

type voidViewer struct {
}

// VoidViewer creates and returns a no-op documentation viewer.
//
// Use this when you want to disable route documentation while keeping
// the Router’s DocViewer interface satisfied.
func VoidViewer() IDocViewer {
	return &voidViewer{}
}

// Handlers returns an empty slice since this viewer does not expose any handlers.
func (v *voidViewer) Handlers() []DocViewerHandler {
	return make([]DocViewerHandler, 0)
}

// RegisterRoute does nothing and returns the viewer itself.
func (v *voidViewer) RegisterRoute(route DocOperation) IDocViewer {
	return v
}

// RegisterGroup does nothing and returns the viewer itself.
func (v *voidViewer) RegisterGroup(group string, data DocGroup) IDocViewer {
	return v
}
