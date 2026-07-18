package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	connect "connectrpc.com/connect"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/authz"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

const (
	uploadCapability       = "import.write"
	uploadMultipartMaxMem  = 32 << 20
	uploadRoutePathPattern = "POST /upload/{raceGridId}"
)

//nolint:lll // readability
type currentPrincipalResolver interface {
	CurrentPrincipalFromRequest(ctx context.Context, r *http.Request) (authn.Principal, bool, error)
}

type uploadResultsService interface {
	UploadResultsFile(
		ctx context.Context,
		req *connect.Request[importv1.UploadResultsFileRequest],
	) (*connect.Response[importv1.UploadResultsFileResponse], error)
}

type multipartUploadHandler struct {
	logger            *log.Logger
	principalResolver currentPrincipalResolver
	authorizer        *authz.CapabilityAuthorizer
	repo              rootrepo.Repository
	service           uploadResultsService
}
type importData struct {
	format  commonv1.ImportFormat
	payload []byte
}

//nolint:whitespace // multiline signature for line-length compliance
func registerMultipartUploadHandler(
	mux *http.ServeMux,
	logger *log.Logger,
	principalResolver currentPrincipalResolver,
	authorizer *authz.CapabilityAuthorizer,
	repo rootrepo.Repository,
	service uploadResultsService,
) {
	handler := &multipartUploadHandler{
		logger:            logger.Named("multipart_upload"),
		principalResolver: principalResolver,
		authorizer:        authorizer,
		repo:              repo,
		service:           service,
	}

	mux.HandleFunc(uploadRoutePathPattern, handler.handleUpload)
}

//nolint:govet,funlen // error handling by design
func (h *multipartUploadHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	principal, found, err := h.principalResolver.CurrentPrincipalFromRequest(
		r.Context(), r)
	if err != nil {
		h.logger.WithCtx(r.Context()).Warn("resolve principal from request",
			log.ErrorField(err))
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}
	if !found {
		http.Error(w, "missing authentication credentials", http.StatusUnauthorized)
		return
	}

	raceGridID, err := parseRaceGridID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	seriesID, err := h.resolveSeriesID(r.Context(), raceGridID)
	if err != nil {
		h.logger.WithCtx(r.Context()).Warn("resolve scope for upload", log.ErrorField(err))
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, rootrepo.ErrNotFound) {
			http.Error(w, "race grid not found", http.StatusNotFound)
			return
		}
		http.Error(w, "unable to resolve race grid scope", http.StatusInternalServerError)
		return
	}

	if err := h.authorizer.Authorize(
		r.Context(), &principal, uploadCapability, authz.ResourceScope{
			SeriesID: strconv.FormatInt(int64(seriesID), 10),
		}); err != nil {
		h.logger.WithCtx(r.Context()).Warn("upload authorization denied",
			log.ErrorField(err))
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	uploads, err := parseMultipartUploadRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	processingStates := make([]any, 0, len(uploads))
	for i := range uploads {
		upload := uploads[i]
		resp, serviceErr := h.service.UploadResultsFile(
			authn.AddPrincipal(r.Context(), &principal),
			connect.NewRequest(&importv1.UploadResultsFileRequest{
				RaceGridId:   uint32(raceGridID),
				ImportFormat: upload.format,
				Payload:      upload.payload,
			}),
		)
		if serviceErr != nil {
			writeUploadServiceError(w, serviceErr)
			return
		}

		processingStates = append(processingStates, resp.Msg.GetProcessingState())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{
		"raceGridId":       raceGridID,
		"uploadedParts":    len(uploads),
		"processingStates": processingStates,
	}); encodeErr != nil {
		h.logger.WithCtx(r.Context()).Warn("encode upload response",
			log.ErrorField(encodeErr))
	}
}

//nolint:whitespace // editor/linter issue
func (h *multipartUploadHandler) resolveSeriesID(
	ctx context.Context,
	raceGridID int32,
) (int32, error) {
	event, err := h.repo.Events().LoadByGridID(ctx, raceGridID)
	if err != nil {
		return 0, err
	}

	season, err := h.repo.Seasons().LoadByID(ctx, event.SeasonID)
	if err != nil {
		return 0, err
	}

	return season.SeriesID, nil
}

func parseRaceGridID(r *http.Request) (int32, error) {
	raw := strings.TrimSpace(r.PathValue("raceGridId"))
	if raw == "" {
		return 0, fmt.Errorf("raceGridId path parameter is required")
	}

	gridID64, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || gridID64 <= 0 {
		return 0, fmt.Errorf("raceGridId must be a positive integer")
	}

	return int32(gridID64), nil
}

//nolint:whitespace,funlen // editor/linter issue, much to do
func parseMultipartUploadRequest(r *http.Request) (
	[]importData, error,
) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("invalid multipart form data")
	}

	uploads := make([]importData, 0, 1)
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("read multipart part: %w", nextErr)
		}

		// Skip non-file form fields and only process upload-like parts.
		if strings.TrimSpace(part.FileName()) == "" &&
			strings.TrimSpace(part.FormName()) != "file" &&
			strings.TrimSpace(part.FormName()) != "payload" {
			part.Close()
			continue
		}

		format, detectErr := detectImportFormatFromContentType(
			part.Header.Get("Content-Type"),
		)
		if detectErr != nil {
			part.Close()
			return nil, detectErr
		}

		payload, readErr := io.ReadAll(part)
		part.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read upload payload: %w", readErr)
		}
		if len(payload) == 0 {
			return nil, fmt.Errorf("payload is required")
		}

		uploads = append(uploads, importData{format: format, payload: payload})
	}

	if len(uploads) == 0 {
		return nil, fmt.Errorf("file form field is required")
	}

	return uploads, nil
}

//nolint:lll // readability
func detectImportFormatFromContentType(contentType string) (commonv1.ImportFormat, error) {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return 0, fmt.Errorf("file content type is required")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return 0, fmt.Errorf("invalid file content type %q", contentType)
	}

	return detectImportFormatFromMediaType(mediaType)
}

func detectImportFormatFromMediaType(mediaType string) (commonv1.ImportFormat, error) {
	normalized := strings.ToLower(strings.TrimSpace(mediaType))
	switch {
	case normalized == "application/json",
		normalized == "text/json",
		strings.HasSuffix(normalized, "+json"):
		return commonv1.ImportFormat_IMPORT_FORMAT_JSON, nil
	case normalized == "text/csv", normalized == "application/csv":
		return commonv1.ImportFormat_IMPORT_FORMAT_CSV, nil
	case normalized == "application/xml",
		normalized == "text/xml",
		strings.HasSuffix(normalized, "+xml"):
		return commonv1.ImportFormat_IMPORT_FORMAT_XML, nil
	default:
		return 0, fmt.Errorf("unsupported file content type %q", mediaType)
	}
}

func writeUploadServiceError(w http.ResponseWriter, err error) {
	connectErr := new(connect.Error)
	if !errors.As(err, &connectErr) {
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	message := connectErr.Message()
	if message == "" {
		message = "upload failed"
	}
	//nolint:exhaustive // only map expected error codes, default to 500 for others
	switch connectErr.Code() {
	case connect.CodeInvalidArgument:
		http.Error(w, message, http.StatusBadRequest)
	case connect.CodeUnauthenticated:
		http.Error(w, message, http.StatusUnauthorized)
	case connect.CodePermissionDenied:
		http.Error(w, message, http.StatusForbidden)
	case connect.CodeNotFound:
		http.Error(w, message, http.StatusNotFound)
	case connect.CodeFailedPrecondition:
		http.Error(w, message, http.StatusPreconditionFailed)
	default:
		http.Error(w, message, http.StatusInternalServerError)
	}
}
