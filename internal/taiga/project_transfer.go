package taiga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"time"
)

type ProjectExportResult struct {
	ExportID string `json:"export_id,omitempty"`
	URL      string `json:"url,omitempty"`
}

func (result ProjectExportResult) Accepted() bool { return result.ExportID != "" && result.URL == "" }

type ProjectImportResult struct {
	Project
	ImportID string `json:"import_id,omitempty"`
}

func (result ProjectImportResult) Accepted() bool {
	return result.ImportID != "" && result.ID == 0
}

func (c *Client) RequestProjectExport(ctx context.Context, projectID int64, dumpFormat string) (ProjectExportResult, error) {
	query := url.Values{"dump_format": []string{dumpFormat}}
	var result ProjectExportResult
	_, err := c.GetOnce(ctx, fmt.Sprintf("exporter/%d", projectID), query, &result)
	return result, err
}

func (c *Client) ImportProjectDump(ctx context.Context, name, contentType string, source io.Reader) (ProjectImportResult, error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	writeDone := make(chan error, 1)
	go func() {
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="dump"; filename=%q`, name))
		partHeader.Set("Content-Type", contentType)
		part, err := multipartWriter.CreatePart(partHeader)
		if err == nil {
			_, err = io.Copy(part, source)
		}
		if err == nil {
			err = multipartWriter.Close()
		}
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			writeDone <- err
			return
		}
		writeDone <- pipeWriter.Close()
	}()

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "importer/load_dump"})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), pipeReader)
	if err != nil {
		_ = pipeReader.Close()
		return ProjectImportResult{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	request.Header.Set("User-Agent", "taiga-cli/0.1")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	started := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		return ProjectImportResult{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + endpoint.Path, Message: "project import may have been accepted; verify before retrying", Retryable: false, Cause: err}
	}
	c.log(http.MethodPost, endpoint.Path, response.StatusCode, time.Since(started))
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	closeErr := response.Body.Close()
	writeErr := <-writeDone
	if writeErr != nil {
		return ProjectImportResult{}, &Error{Kind: KindTransport, Operation: "POST " + endpoint.Path, Message: "stream project dump", Retryable: false, Cause: writeErr}
	}
	if readErr != nil || closeErr != nil {
		if readErr == nil {
			readErr = closeErr
		}
		return ProjectImportResult{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + endpoint.Path, Message: "project import may have been accepted; verify before retrying", Retryable: false, Cause: readErr}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ProjectImportResult{}, decodeAPIError("POST "+endpoint.Path, response.StatusCode, data)
	}
	var result ProjectImportResult
	if err := json.Unmarshal(data, &result); err != nil {
		return ProjectImportResult{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + endpoint.Path, Message: "project import was accepted but its result could not be decoded", Retryable: false, Cause: err}
	}
	return result, nil
}
