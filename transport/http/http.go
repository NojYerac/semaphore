package http

import (
	json "encoding/json"
	nethttp "net/http"

	"github.com/go-playground/validator/v10"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/go-lib/tracing"
	"github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/semaphore/data"
)

func RegisterRoutes(src data.DataEngine, srv http.Server) {
	v := validator.New()
	t := tracing.TracerForPackage()

	srv.HandleFunc("GET /flags", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx, span := t.Start(r.Context(), "ListFlagsHandler")
		defer span.End()
		logger := log.FromContext(ctx)
		flags, err := src.GetFlags(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to get flags")
			nethttp.Error(w, "failed to get flags", nethttp.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(flags); err != nil {
			logger.WithError(err).Error("failed to encode flags")
			nethttp.Error(w, "failed to encode flags", nethttp.StatusInternalServerError)
			return
		}
	})

	srv.HandleFunc("POST /flags", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx, span := t.Start(r.Context(), "CreateFlagHandler")
		defer span.End()
		logger := log.FromContext(ctx)
		contentType := r.Header.Get("Content-Type")
		if len(contentType) < 16 || contentType[0:16] != "application/json" {
			nethttp.Error(w, "unsupported content type", nethttp.StatusUnsupportedMediaType)
			return
		}
		flag := new(data.FeatureFlag)
		if err := json.NewDecoder(r.Body).Decode(flag); err != nil {
			logger.WithError(err).Error("failed to decode flag")
			nethttp.Error(w, "failed to decode flag", nethttp.StatusBadRequest)
			return
		}
		if err := v.Struct(flag); err != nil {
			logger.WithError(err).Error("invalid flag")
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}
		id, err := src.CreateFlag(ctx, flag)
		if err != nil {
			logger.WithError(err).Error("failed to create flag")
			nethttp.Error(w, "failed to create flag", nethttp.StatusInternalServerError)
			return
		}
		w.WriteHeader(nethttp.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"id": id}); err != nil {
			logger.WithError(err).Error("failed to encode response")
			nethttp.Error(w, "failed to encode response", nethttp.StatusInternalServerError)
			return
		}
	})

	srv.HandleFunc("GET /flags/{id}", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx, span := t.Start(r.Context(), "GetFlagHandler")
		defer span.End()
		logger := log.FromContext(ctx)
		id := r.PathValue("id")
		flag, err := src.GetFlagByID(ctx, id)
		if err != nil {
			logger.WithError(err).Error("failed to get flag")
			nethttp.Error(w, "failed to get flag", nethttp.StatusInternalServerError)
			return
		}
		if flag == nil {
			nethttp.Error(w, "flag not found", nethttp.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(flag); err != nil {
			logger.WithError(err).Error("failed to encode flag")
			nethttp.Error(w, "failed to encode flag", nethttp.StatusInternalServerError)
			return
		}
	})

	srv.HandleFunc("PUT /flags/{id}", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx, span := t.Start(r.Context(), "UpdateFlagHandler")
		defer span.End()
		logger := log.FromContext(ctx)
		id := r.PathValue("id")
		contentType := r.Header.Get("Content-Type")
		if len(contentType) < 16 || contentType[0:16] != "application/json" {
			nethttp.Error(w, "unsupported content type", nethttp.StatusUnsupportedMediaType)
			return
		}
		flag := new(data.FeatureFlag)
		if err := json.NewDecoder(r.Body).Decode(flag); err != nil {
			logger.WithError(err).Error("failed to decode flag")
			nethttp.Error(w, "failed to decode flag", nethttp.StatusBadRequest)
			return
		}
		flag.ID = id
		if err := v.Struct(flag); err != nil {
			logger.WithError(err).Error("invalid flag")
			nethttp.Error(w, err.Error(), nethttp.StatusUnprocessableEntity)
			return
		}
		if err := src.UpdateFlag(ctx, flag); err != nil {
			logger.WithError(err).Error("failed to update flag")
			nethttp.Error(w, "failed to update flag", nethttp.StatusInternalServerError)
			return
		}
		w.WriteHeader(nethttp.StatusOK)
		if err := json.NewEncoder(w).Encode(flag); err != nil {
			logger.WithError(err).Error("failed to encode flag")
			nethttp.Error(w, "failed to encode flag", nethttp.StatusInternalServerError)
			return
		}
	})

	srv.HandleFunc("DELETE /flags/{id}", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx, span := t.Start(r.Context(), "DeleteFlagHandler")
		defer span.End()
		logger := log.FromContext(ctx)
		id := r.PathValue("id")
		if err := src.DeleteFlag(ctx, id); err != nil {
			logger.WithError(err).Error("failed to delete flag")
			nethttp.Error(w, "failed to delete flag", nethttp.StatusInternalServerError)
			return
		}
		w.WriteHeader(nethttp.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
			logger.WithError(err).Error("failed to encode response")
			nethttp.Error(w, "failed to encode response", nethttp.StatusInternalServerError)
			return
		}
	})

	type evaluateInput struct {
		UserID   string   `json:"userID" validate:"required,uuid4"`
		GroupIDs []string `json:"groupIDs" validate:"required,dive,uuid4"`
	}

	srv.HandleFunc("POST /flags/{id}/evaluate", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		ctx, span := t.Start(r.Context(), "EvaluateFlagHandler")
		defer span.End()
		logger := log.FromContext(ctx)
		id := r.PathValue("id")
		contentType := r.Header.Get("Content-Type")
		if len(contentType) < 16 || contentType[0:16] != "application/json" {
			nethttp.Error(w, "unsupported content type", nethttp.StatusUnsupportedMediaType)
			return
		}
		input := new(evaluateInput)
		if err := json.NewDecoder(r.Body).Decode(input); err != nil {
			logger.WithError(err).Error("failed to decode input")
			nethttp.Error(w, "failed to decode input", nethttp.StatusBadRequest)
			return
		}
		result, err := src.EvaluateFlag(ctx, id, input.UserID, input.GroupIDs)
		if err != nil {
			logger.WithError(err).Error("failed to evaluate flag")
			nethttp.Error(w, "failed to evaluate flag", nethttp.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"result": result}); err != nil {
			logger.WithError(err).Error("failed to encode result")
			nethttp.Error(w, "failed to encode result", nethttp.StatusInternalServerError)
			return
		}
	})
}
