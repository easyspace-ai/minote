package handlers

import (
	"net/http"
)

// Adapter provides helper methods to integrate handlers with the existing Server.
type Adapter struct {
	modelHandler *ModelHandler
}

// NewAdapter creates a new handler adapter.
func NewAdapter(defaultModel string) *Adapter {
	return &Adapter{
		modelHandler: NewModelHandler(defaultModel),
	}
}

// SetDefaultModel updates the default model for all handlers.
func (a *Adapter) SetDefaultModel(model string) {
	a.modelHandler.SetDefaultModel(model)
}

// ModelHandler returns the model handler.
func (a *Adapter) ModelHandler() *ModelHandler {
	return a.modelHandler
}

// HandleModelsList wraps ModelHandler.HandleList for compatibility.
func (a *Adapter) HandleModelsList(w http.ResponseWriter, r *http.Request) {
	a.modelHandler.HandleList(w, r)
}

// HandleModelGet wraps ModelHandler.HandleGet for compatibility.
func (a *Adapter) HandleModelGet(w http.ResponseWriter, r *http.Request) {
	a.modelHandler.HandleGet(w, r)
}

// FindModelByNameOrID is a convenience method to find a model.
func (a *Adapter) FindModelByNameOrID(modelName string) (any, bool) {
	return a.modelHandler.FindModelByNameOrID(modelName)
}
