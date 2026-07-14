package server

import (
	"net/http"
	"strings"
)

func (a *App) handleDebugSessionPath(writer http.ResponseWriter, request *http.Request) {
	if strings.HasSuffix(request.URL.Path, "/commands") {
		if request.Method != http.MethodPost {
			writeAPIError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		a.handleDebugCommand(writer, request)
		return
	}
	if request.Method == http.MethodDelete {
		a.handleDeleteDebugSession(writer, request)
		return
	}
	writeAPIError(writer, http.StatusNotFound, "not_found", "debug endpoint not found")
}

func (a *App) handleCreateDebugSession(writer http.ResponseWriter, request *http.Request) {
	var input DebugSessionRequest
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	snapshot, err := a.debug.Create(request.Context(), input)
	if err != nil {
		writeAPIError(writer, http.StatusServiceUnavailable, "debug_start_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusCreated, snapshot)
}

func (a *App) handleDebugCommand(writer http.ResponseWriter, request *http.Request) {
	id := debugSessionID(request.URL.Path)
	var input struct {
		Command string `json:"command"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	snapshot, err := a.debug.Command(request.Context(), id, input.Command)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "debug_command_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, snapshot)
}

func (a *App) handleDeleteDebugSession(writer http.ResponseWriter, request *http.Request) {
	id := debugSessionID(request.URL.Path)
	if err := a.debug.Delete(request.Context(), id); err != nil {
		writeAPIError(writer, http.StatusNotFound, "debug_session_not_found", err.Error())
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func debugSessionID(path string) string {
	remainder := strings.TrimPrefix(path, "/api/debug/sessions/")
	remainder = strings.TrimSuffix(remainder, "/commands")
	return strings.Trim(remainder, "/")
}
