package taiga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func attachmentPath(resource string) (string, error) {
	switch resource {
	case "issue":
		return "issues/attachments", nil
	case "story":
		return "userstories/attachments", nil
	case "task":
		return "tasks/attachments", nil
	default:
		return "", fmt.Errorf("unsupported attachment resource %q", resource)
	}
}

func (c *Client) ListAttachments(ctx context.Context, resource string, projectID, objectID int64) ([]Attachment, error) {
	path, err := attachmentPath(resource)
	if err != nil {
		return nil, err
	}
	query := url.Values{"project": []string{strconv.FormatInt(projectID, 10)}, "object_id": []string{strconv.FormatInt(objectID, 10)}, "page_size": []string{"1000"}}
	var attachments []Attachment
	_, err = c.Get(ctx, path, query, &attachments)
	return attachments, err
}

func (c *Client) GetAttachment(ctx context.Context, resource string, id int64) (Attachment, error) {
	path, err := attachmentPath(resource)
	if err != nil {
		return Attachment{}, err
	}
	var attachment Attachment
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d", path, id), nil, &attachment)
	return attachment, err
}

func (c *Client) CreateAttachment(ctx context.Context, resource string, projectID, objectID int64, name, description string, deprecated bool, source io.Reader) (Attachment, error) {
	path, err := attachmentPath(resource)
	if err != nil {
		return Attachment{}, err
	}
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	writeDone := make(chan error, 1)
	go func() {
		defer close(writeDone)
		fields := map[string]string{
			"project":       strconv.FormatInt(projectID, 10),
			"object_id":     strconv.FormatInt(objectID, 10),
			"description":   description,
			"is_deprecated": strconv.FormatBool(deprecated),
			"from_comment":  "false",
		}
		for key, value := range fields {
			if err := multipartWriter.WriteField(key, value); err != nil {
				_ = pipeWriter.CloseWithError(err)
				writeDone <- err
				return
			}
		}
		part, err := multipartWriter.CreateFormFile("attached_file", name)
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

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), pipeReader)
	if err != nil {
		_ = pipeReader.Close()
		return Attachment{}, err
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
		return Attachment{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + endpoint.Path, Message: "attachment may have been uploaded; verify before retrying", Retryable: false, Cause: err}
	}
	c.log(http.MethodPost, endpoint.Path, response.StatusCode, time.Since(started))
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	closeErr := response.Body.Close()
	writeErr := <-writeDone
	if writeErr != nil {
		return Attachment{}, &Error{Kind: KindTransport, Operation: "POST " + endpoint.Path, Message: "stream attachment upload", Retryable: false, Cause: writeErr}
	}
	if readErr != nil || closeErr != nil {
		if readErr == nil {
			readErr = closeErr
		}
		return Attachment{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + endpoint.Path, Message: "attachment may have been uploaded; verify before retrying", Retryable: false, Cause: readErr}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Attachment{}, decodeAPIError("POST "+endpoint.Path, response.StatusCode, data)
	}
	var attachment Attachment
	if err := json.Unmarshal(data, &attachment); err != nil {
		return Attachment{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + endpoint.Path, Message: "attachment was accepted but its result could not be decoded", Retryable: false, Cause: err}
	}
	return attachment, nil
}

func (c *Client) UpdateAttachment(ctx context.Context, resource string, id int64, request UpdateAttachmentRequest) (Attachment, error) {
	path, err := attachmentPath(resource)
	if err != nil {
		return Attachment{}, err
	}
	var attachment Attachment
	_, err = c.Patch(ctx, fmt.Sprintf("%s/%d", path, id), request, &attachment)
	return attachment, err
}

func (c *Client) DeleteAttachment(ctx context.Context, resource string, id int64) error {
	path, err := attachmentPath(resource)
	if err != nil {
		return err
	}
	return c.Delete(ctx, fmt.Sprintf("%s/%d", path, id))
}

func NormalizeAttachmentResource(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "issue", "issues":
		return "issue", nil
	case "story", "stories", "userstory", "userstories", "us":
		return "story", nil
	case "task", "tasks":
		return "task", nil
	default:
		return "", fmt.Errorf("resource must be issue, story, or task")
	}
}
