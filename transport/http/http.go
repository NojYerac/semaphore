package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/nojyerac/go-lib/auth"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/go-lib/tracing"
	libhttp "github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/semaphore/data"
	"go.opentelemetry.io/otel/trace"
)

func RegisterRoutes(src data.DataEngine, srv libhttp.Server) {
	r := &Routes{
		v:   validator.New(),
		t:   tracing.TracerForPackage(),
		src: src,
	}

	srv.HandleFunc("GET /flags", r.GetFlagsHandler)

	srv.HandleFunc("POST /flags", r.CreateFlagHandler)

	srv.HandleFunc("GET /flags/{id}", r.GetFlagHandler)

	srv.HandleFunc("PUT /flags/{id}", r.UpdateFlagHandler)

	srv.HandleFunc("DELETE /flags/{id}", r.DeleteFlagHandler)

	srv.HandleFunc("POST /flags/{id}/evaluate", r.EvaluateFlagHandler)
}

type Routes struct {
	v   *validator.Validate
	t   trace.Tracer
	src data.DataEngine
}

func (r *Routes) GetFlagsHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := r.t.Start(req.Context(), "ListFlagsHandler")
	defer span.End()
	flags, err := r.src.GetFlags(ctx)
	if err != nil {
		r.writeError(ctx, w, err, "failed to get flags", http.StatusInternalServerError)
		return
	}
	r.writeJSON(ctx, w, http.StatusOK, flags)
}

func (r *Routes) CreateFlagHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := r.t.Start(req.Context(), "CreateFlagHandler")
	defer span.End()
	if !r.requireJSONContentType(w, req) {
		return
	}
	flag := new(data.FeatureFlag)
	if !r.decodeJSONBody(ctx, w, req, flag) {
		return
	}
	if err := r.v.Struct(flag); err != nil {
		r.writeError(ctx, w, err, "invalid input", http.StatusBadRequest)
		return
	}
	id, err := r.src.CreateFlag(ctx, flag)
	if err != nil {
		r.writeError(ctx, w, err, "failed to create flag", http.StatusInternalServerError)
		return
	}
	r.writeJSON(ctx, w, http.StatusCreated, map[string]string{"id": id})
}

func (r *Routes) GetFlagHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := r.t.Start(req.Context(), "GetFlagHandler")
	defer span.End()
	id := req.PathValue("id")
	flag, err := r.src.GetFlagByID(ctx, id)
	if err != nil {
		r.writeError(ctx, w, err, "failed to get flag", http.StatusInternalServerError)
		return
	}
	if flag == nil {
		http.Error(w, "flag not found", http.StatusNotFound)
		return
	}
	r.writeJSON(ctx, w, http.StatusOK, flag)
}

func (r *Routes) UpdateFlagHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := r.t.Start(req.Context(), "UpdateFlagHandler")
	defer span.End()
	claims, ok := auth.FromContext(ctx)
	if ok {
		log.FromContext(ctx).WithField("actor", claims.Subject).Info("updating flag")
	}
	id := req.PathValue("id")
	if !r.requireJSONContentType(w, req) {
		return
	}
	flag := new(data.FeatureFlag)
	if !r.decodeJSONBody(ctx, w, req, flag) {
		return
	}
	flag.ID = id
	if err := r.v.Struct(flag); err != nil {
		r.writeError(ctx, w, err, "invalid input", http.StatusUnprocessableEntity)
		return
	}
	if err := r.src.UpdateFlag(ctx, flag); err != nil {
		r.writeError(ctx, w, err, "failed to update flag", http.StatusInternalServerError)
		return
	}
	r.writeJSON(ctx, w, http.StatusOK, flag)
}

func (r *Routes) DeleteFlagHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := r.t.Start(req.Context(), "DeleteFlagHandler")
	defer span.End()
	id := req.PathValue("id")
	if err := r.src.DeleteFlag(ctx, id); err != nil {
		r.writeError(ctx, w, err, "failed to delete flag", http.StatusInternalServerError)
		return
	}
	r.writeJSON(ctx, w, http.StatusOK, map[string]bool{"success": true})
}

func (r *Routes) EvaluateFlagHandler(w http.ResponseWriter, req *http.Request) {
	ctx, span := r.t.Start(req.Context(), "EvaluateFlagHandler")
	defer span.End()
	id := req.PathValue("id")
	if !r.requireJSONContentType(w, req) {
		return
	}
	input := new(evaluateInput)
	if !r.decodeJSONBody(ctx, w, req, input) {
		return
	}
	result, err := r.src.EvaluateFlag(ctx, id, input.UserID, input.GroupIDs)
	if err != nil {
		r.writeError(ctx, w, err, "failed to evaluate flag", http.StatusInternalServerError)
		return
	}
	r.writeJSON(
		ctx,
		w,
		http.StatusOK,
		map[string]interface{}{"result": result},
	)
}

const (
	contentTypeHeader   = "Content-Type"
	mimeApplicationJSON = "application/json"
	failedToDecodeMsg   = "failed to decode input"
	failedToEncodeMsg   = "failed to encode result"
	badContentTypeMsg   = "unsupported content type"
)

type evaluateInput struct {
	UserID   string   `json:"userID" validate:"required,uuid4"`
	GroupIDs []string `json:"groupIDs" validate:"required,dive,uuid4"`
}

func (r *Routes) writeError(
	ctx context.Context, w http.ResponseWriter, err error, message string, status int,
) {
	log.FromContext(ctx).WithError(err).Error(message)
	http.Error(w, message, status)
}

func (r *Routes) requireJSONContentType(w http.ResponseWriter, req *http.Request) bool {
	ct := req.Header.Get(contentTypeHeader)
	if !strings.HasPrefix(ct, mimeApplicationJSON) {
		err := fmt.Errorf("unsuported content type: %q", ct)
		r.writeError(req.Context(), w, err, badContentTypeMsg, http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

func (r *Routes) decodeJSONBody(
	ctx context.Context, w http.ResponseWriter, req *http.Request, target interface{},
) bool {
	if err := json.NewDecoder(req.Body).Decode(target); err != nil {
		r.writeError(ctx, w, err, failedToDecodeMsg, http.StatusBadRequest)
		return false
	}
	return true
}

func (r *Routes) writeJSON(
	ctx context.Context, w http.ResponseWriter, status int, payload interface{},
) {
	w.Header().Set(contentTypeHeader, mimeApplicationJSON)
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		r.writeError(ctx, w, err, failedToEncodeMsg, http.StatusInternalServerError)
		return
	}
}
