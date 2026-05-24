package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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

	format, payload, err := parseMultipartUploadRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.UploadResultsFile(
		authn.AddPrincipal(r.Context(), &principal),
		connect.NewRequest(&importv1.UploadResultsFileRequest{
			RaceGridId:   uint32(raceGridID),
			ImportFormat: format,
			Payload:      payload,
		}),
	)
	if err != nil {
		writeUploadServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{
		"raceGridId":      resp.Msg.GetRaceGridId(),
		"processingState": resp.Msg.GetProcessingState(),
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

//nolint:whitespace // editor/linter issue
func parseMultipartUploadRequest(r *http.Request) (
	commonv1.ImportFormat, []byte, error,
) {
	if err := r.ParseMultipartForm(uploadMultipartMaxMem); err != nil {
		return 0, nil, fmt.Errorf("invalid multipart form data")
	}

	format, file, err := openUploadPayload(r)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	payload, err := io.ReadAll(file)
	if err != nil {
		return 0, nil, fmt.Errorf("read upload payload: %w", err)
	}
	if len(payload) == 0 {
		return 0, nil, fmt.Errorf("payload is required")
	}

	return format, payload, nil
}

func openUploadPayload(r *http.Request) (commonv1.ImportFormat, multipartFile, error) {
	for _, field := range []string{"file", "payload"} {
		file, header, err := r.FormFile(field)
		if err == nil {
			format, detectErr := detectImportFormatFromPartHeader(header)
			if detectErr != nil {
				file.Close()
				return 0, nil, detectErr
			}
			return format, file, nil
		}
		if !errors.Is(err, http.ErrMissingFile) {
			return 0, nil, fmt.Errorf("read multipart file %q: %w", field, err)
		}
	}

	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		return 0, nil, fmt.Errorf("file form field is required")
	}

	for _, headers := range r.MultipartForm.File {
		if len(headers) == 0 {
			continue
		}
		header := headers[0]
		file, err := header.Open()
		if err != nil {
			return 0, nil, fmt.Errorf("open multipart file: %w", err)
		}

		format, detectErr := detectImportFormatFromPartHeader(header)
		if detectErr != nil {
			file.Close()
			return 0, nil, detectErr
		}

		return format, file, nil
	}

	return 0, nil, fmt.Errorf("file form field is required")
}

//nolint:whitespace // editor/linter issue
func detectImportFormatFromPartHeader(
	header *multipart.FileHeader,
) (commonv1.ImportFormat, error) {
	if header == nil {
		return 0, fmt.Errorf("file content type is required")
	}

	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
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

type multipartFile interface {
	io.Reader
	io.Closer
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
