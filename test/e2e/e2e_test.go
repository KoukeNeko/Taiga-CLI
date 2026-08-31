//go:build integration

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type envelope struct {
	Data  map[string]any   `json:"data"`
	Items []map[string]any `json:"items"`
	Plan  map[string]any   `json:"plan"`
}

func TestPhaseOneAgainstDocker(t *testing.T) {
	baseURL := requiredEnv(t, "TAIGA_E2E_URL")
	host := requiredEnv(t, "TAIGA_E2E_HOST")
	binary := requiredEnv(t, "TAIGA_E2E_BIN")
	home := t.TempDir()
	username := "e2e_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	password := "E2E-Password-7fK2mQ9"
	token := register(t, baseURL, username, password)
	project := createProject(t, baseURL, token)
	projectID := int64(project["id"].(float64))
	projectSlug := project["slug"].(string)
	t.Cleanup(func() {
		apiRequest(t, http.MethodDelete, baseURL+"projects/"+strconv.FormatInt(projectID, 10), token, nil, nil)
	})

	runner := cliRunner{t: t, binary: binary, dir: t.TempDir(), env: []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"TAIGA_API_URL=" + baseURL,
		"TAIGA_TOKEN=" + token,
		"TAIGA_PROJECT=" + projectSlug,
	}}

	runner.jsonOK("doctor", "--host", host)
	runner.jsonOK("auth", "status")
	runner.jsonOK("project", "list", "--limit", "10")
	runner.jsonOK("project", "view", projectSlug)
	runner.jsonOK("project", "use", projectSlug)
	runner.jsonOK("schema", "issue", "view")

	created := runner.jsonOK("issue", "create", "--subject", "E2E issue", "--description", "created by integration test")
	ref := int(created.Data["ref"].(float64))
	version := int(created.Data["version"].(float64))
	target := strconv.Itoa(ref)

	listed := runner.jsonOK("issue", "list", "--fields", "ref,subject,status,version")
	if !containsRef(listed.Items, ref) {
		t.Fatalf("created issue ref %d missing from list", ref)
	}
	runner.jsonOK("issue", "view", target, "--fields", "ref,subject,version")

	dryRun := runner.jsonOK("issue", "edit", target, "--subject", "must not persist", "--dry-run")
	if dryRun.Plan["performed"] != false || dryRun.Plan["would_write"] != true {
		t.Fatalf("dry-run plan = %#v", dryRun.Plan)
	}
	view := runner.jsonOK("issue", "view", target, "--fields", "subject")
	if view.Data["subject"] != "E2E issue" {
		t.Fatalf("dry-run mutated issue: %#v", view.Data)
	}

	edited := runner.jsonOK("issue", "edit", target, "--subject", "E2E issue updated", "--base-version", strconv.Itoa(version))
	version = int(edited.Data["version"].(float64))
	issueID := int64(edited.Data["id"].(float64))
	var externalUpdate map[string]any
	apiRequest(t, http.MethodPatch, baseURL+"issues/"+strconv.FormatInt(issueID, 10), token, map[string]any{"subject": "external concurrent edit", "version": version}, &externalUpdate)
	externalVersion := int(externalUpdate["version"].(float64))
	stdout, stderr, code := runner.run("--json", "issue", "edit", target, "--subject", "must conflict", "--base-version", strconv.Itoa(version))
	if code != 6 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("stale OCC edit exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	var conflictEnvelope map[string]any
	if err := json.Unmarshal([]byte(stderr), &conflictEnvelope); err != nil || conflictEnvelope["error"].(map[string]any)["code"] != "occ_conflict" {
		t.Fatalf("invalid OCC error: %v: %s", err, stderr)
	}
	afterConflict := runner.jsonOK("issue", "view", target, "--fields", "subject,version")
	if afterConflict.Data["subject"] != "external concurrent edit" {
		t.Fatalf("stale edit overwrote concurrent change: %#v", afterConflict.Data)
	}
	version = externalVersion
	assigned := runner.jsonOK("issue", "assign", target, "--to", username, "--base-version", strconv.Itoa(version))
	version = int(assigned.Data["version"].(float64))
	commented := runner.jsonOK("issue", "comment", target, "--body", "integration comment", "--base-version", strconv.Itoa(version))
	version = int(commented.Data["version"].(float64))
	watchedIssue := runner.jsonOK("issue", "watch", target)
	if watchedIssue.Data["watching"] != true || watchedIssue.Data["verified"] != true {
		t.Fatalf("issue watch was not verified: %#v", watchedIssue.Data)
	}
	issueView := runner.jsonOK("issue", "view", target, "--fields", "ref,is_watcher")
	if issueView.Data["is_watcher"] != true {
		t.Fatalf("issue view did not confirm watch state: %#v", issueView.Data)
	}
	issueHistory := runner.jsonOK("issue", "history", target, "--type", "comment", "--fields", "id,kind,author,comment")
	if !containsComment(issueHistory.Items, "integration comment") {
		t.Fatalf("issue comment missing from CLI history: %#v", issueHistory.Items)
	}
	unwatchedIssue := runner.jsonOK("issue", "unwatch", target)
	if unwatchedIssue.Data["watching"] != false || unwatchedIssue.Data["verified"] != true {
		t.Fatalf("issue unwatch was not verified: %#v", unwatchedIssue.Data)
	}
	closedStatus := firstClosedStatus(t, baseURL, token, projectID)
	closed := runner.jsonOK("issue", "close", target, "--status", closedStatus, "--base-version", strconv.Itoa(version))
	if closed.Data["is_closed"] != true {
		t.Fatalf("issue not closed: %#v", closed.Data)
	}
	attachmentPath := filepath.Join(runner.dir, "evidence.txt")
	if err := os.WriteFile(attachmentPath, []byte("Taiga CLI attachment evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	attachment := runner.jsonOK("attachment", "add", "issue", target, attachmentPath, "--description", "E2E evidence")
	attachmentID := int64(attachment.Data["id"].(float64))
	attachmentList := runner.jsonOK("attachment", "list", "issue", target, "--fields", "id,name,size,description")
	if !containsID(attachmentList.Items, attachmentID) {
		t.Fatalf("attachment %d missing from list", attachmentID)
	}
	runner.jsonOK("attachment", "view", "issue", strconv.FormatInt(attachmentID, 10), "--fields", "id,name,size,url,sha1")
	editedAttachment := runner.jsonOK("attachment", "edit", "issue", strconv.FormatInt(attachmentID, 10), "--description", "Archived evidence", "--deprecated")
	if editedAttachment.Data["description"] != "Archived evidence" || editedAttachment.Data["is_deprecated"] != true {
		t.Fatalf("attachment metadata not updated: %#v", editedAttachment.Data)
	}
	stdout, stderr, code = runner.run("--json", "--no-input", "attachment", "delete", "issue", strconv.FormatInt(attachmentID, 10))
	if code != 10 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("unconfirmed delete exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	deletedAttachment := runner.jsonOK("attachment", "delete", "issue", strconv.FormatInt(attachmentID, 10), "--yes")
	if deletedAttachment.Data["deleted"] != true {
		t.Fatalf("attachment not deleted: %#v", deletedAttachment.Data)
	}
	attachmentList = runner.jsonOK("attachment", "list", "issue", target, "--fields", "id,name")
	if containsID(attachmentList.Items, attachmentID) {
		t.Fatalf("deleted attachment %d remains in list", attachmentID)
	}

	sprint := runner.jsonOK("sprint", "create", "--name", "E2E Sprint", "--start", "2026-08-31", "--finish", "2026-09-07")
	milestoneSlug := sprint.Data["slug"].(string)
	runner.jsonOK("sprint", "view", milestoneSlug, "--fields", "name,slug,start,finish,closed")
	openSprints := runner.jsonOK("sprint", "list", "--state", "open", "--fields", "name,slug,start,finish,closed")
	if !containsSlug(openSprints.Items, milestoneSlug) {
		t.Fatalf("created sprint %q missing from open list", milestoneSlug)
	}
	sprintDryRun := runner.jsonOK("sprint", "edit", milestoneSlug, "--finish", "2026-09-08", "--dry-run")
	if sprintDryRun.Plan["performed"] != false || sprintDryRun.Plan["would_write"] != true {
		t.Fatalf("sprint dry-run plan = %#v", sprintDryRun.Plan)
	}
	story := runner.jsonOK("story", "create", "--subject", "E2E story", "--description", "created by integration test")
	storyRef := int(story.Data["ref"].(float64))
	storyID := int64(story.Data["id"].(float64))
	storyVersion := int(story.Data["version"].(float64))
	storyTarget := strconv.Itoa(storyRef)
	stories := runner.jsonOK("story", "list", "--order-by", "subject", "--fields", "ref,subject,status,version")
	if !containsRef(stories.Items, storyRef) {
		t.Fatalf("created story ref %d missing from list", storyRef)
	}
	runner.jsonOK("story", "view", storyTarget, "--fields", "ref,subject,version,sprint_slug")
	storyDryRun := runner.jsonOK("story", "edit", storyTarget, "--subject", "must not persist", "--dry-run")
	if storyDryRun.Plan["performed"] != false || storyDryRun.Plan["would_write"] != true {
		t.Fatalf("story dry-run plan = %#v", storyDryRun.Plan)
	}
	storyView := runner.jsonOK("story", "view", storyTarget, "--fields", "subject")
	if storyView.Data["subject"] != "E2E story" {
		t.Fatalf("story dry-run mutated state: %#v", storyView.Data)
	}
	storyEdited := runner.jsonOK("story", "edit", storyTarget, "--subject", "E2E story updated", "--base-version", strconv.Itoa(storyVersion))
	storyVersion = int(storyEdited.Data["version"].(float64))
	var externalStoryUpdate map[string]any
	apiRequest(t, http.MethodPatch, baseURL+"userstories/"+strconv.FormatInt(storyID, 10), token, map[string]any{"subject": "external story edit", "version": storyVersion}, &externalStoryUpdate)
	externalStoryVersion := int(externalStoryUpdate["version"].(float64))
	stdout, stderr, code = runner.run("--json", "story", "edit", storyTarget, "--subject", "must conflict", "--base-version", strconv.Itoa(storyVersion))
	if code != 6 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("stale story OCC edit exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	staleDifferentField := runner.jsonOK("story", "edit", storyTarget, "--description", "field-aware stale merge", "--base-version", strconv.Itoa(storyVersion))
	storyVersion = int(staleDifferentField.Data["version"].(float64))
	if storyVersion <= externalStoryVersion {
		t.Fatalf("field-aware stale update did not advance version: %#v", staleDifferentField.Data)
	}
	storyAssigned := runner.jsonOK("story", "assign", storyTarget, "--to", username, "--base-version", strconv.Itoa(storyVersion))
	storyVersion = int(storyAssigned.Data["version"].(float64))
	assignedUsers := storyAssigned.Data["assigned_users"].([]any)
	if len(assignedUsers) != 1 {
		t.Fatalf("story assigned_users = %#v", assignedUsers)
	}
	storyMoved := runner.jsonOK("story", "move", storyTarget, "--sprint", milestoneSlug, "--base-version", strconv.Itoa(storyVersion))
	storyVersion = int(storyMoved.Data["version"].(float64))
	if storyMoved.Data["sprint_slug"] != milestoneSlug {
		t.Fatalf("story sprint = %#v", storyMoved.Data)
	}
	sprintStories := runner.jsonOK("story", "list", "--sprint", milestoneSlug, "--fields", "ref,subject,sprint_slug")
	if !containsRef(sprintStories.Items, storyRef) {
		t.Fatalf("story missing from sprint-filtered list")
	}
	storyBacklog := runner.jsonOK("story", "move", storyTarget, "--sprint", "backlog", "--base-version", strconv.Itoa(storyVersion))
	storyVersion = int(storyBacklog.Data["version"].(float64))
	if value, ok := storyBacklog.Data["sprint_slug"]; ok && value != "" {
		t.Fatalf("story did not return to backlog: %#v", storyBacklog.Data)
	}
	backlogStories := runner.jsonOK("story", "list", "--sprint", "backlog", "--fields", "ref,subject,sprint_slug")
	if !containsRef(backlogStories.Items, storyRef) {
		t.Fatalf("story missing from backlog-filtered list")
	}
	storyComment := runner.jsonOK("story", "comment", storyTarget, "--body", "story integration comment", "--base-version", strconv.Itoa(storyVersion))
	storyVersion = int(storyComment.Data["version"].(float64))
	watchedStory := runner.jsonOK("story", "watch", storyTarget)
	if watchedStory.Data["watching"] != true || watchedStory.Data["verified"] != true {
		t.Fatalf("story watch was not verified: %#v", watchedStory.Data)
	}
	storyHistory := runner.jsonOK("story", "history", storyTarget, "--type", "comment", "--fields", "id,kind,author,comment")
	if !containsComment(storyHistory.Items, "story integration comment") {
		t.Fatalf("story comment missing from CLI history: %#v", storyHistory.Items)
	}
	unwatchedStory := runner.jsonOK("story", "unwatch", storyTarget)
	if unwatchedStory.Data["watching"] != false || unwatchedStory.Data["verified"] != true {
		t.Fatalf("story unwatch was not verified: %#v", unwatchedStory.Data)
	}
	closedStoryStatus := firstClosedStoryStatus(t, baseURL, token, projectID)
	closedStory := runner.jsonOK("story", "close", storyTarget, "--status", closedStoryStatus, "--base-version", strconv.Itoa(storyVersion))
	if closedStory.Data["is_closed"] != true {
		t.Fatalf("story not closed: %#v", closedStory.Data)
	}

	storyWithTask := runner.jsonOK("story", "create", "--subject", "Story with open task", "--sprint", milestoneSlug)
	storyWithTaskRef := int(storyWithTask.Data["ref"].(float64))
	storyWithTaskVersion := int(storyWithTask.Data["version"].(float64))
	task := runner.jsonOK("task", "create", "--story", strconv.Itoa(storyWithTaskRef), "--subject", "E2E task")
	taskRef := int(task.Data["ref"].(float64))
	taskVersion := int(task.Data["version"].(float64))
	taskTarget := strconv.Itoa(taskRef)
	if task.Data["story_ref"] != float64(storyWithTaskRef) || task.Data["sprint_slug"] != milestoneSlug {
		t.Fatalf("task did not inherit parent Story/Sprint: %#v", task.Data)
	}
	tasks := runner.jsonOK("task", "list", "--story", strconv.Itoa(storyWithTaskRef), "--fields", "ref,subject,status,story_ref,version")
	if !containsRef(tasks.Items, taskRef) {
		t.Fatalf("created task ref %d missing from parent-filtered list", taskRef)
	}
	runner.jsonOK("task", "view", taskTarget, "--fields", "ref,subject,status,story_ref,version")
	taskDryRun := runner.jsonOK("task", "edit", taskTarget, "--subject", "must not persist", "--dry-run")
	if taskDryRun.Plan["performed"] != false || taskDryRun.Plan["would_write"] != true {
		t.Fatalf("task dry-run plan = %#v", taskDryRun.Plan)
	}
	taskEdited := runner.jsonOK("task", "edit", taskTarget, "--subject", "E2E task updated", "--base-version", strconv.Itoa(taskVersion))
	taskVersion = int(taskEdited.Data["version"].(float64))
	taskAssigned := runner.jsonOK("task", "assign", taskTarget, "--to", username, "--base-version", strconv.Itoa(taskVersion))
	taskVersion = int(taskAssigned.Data["version"].(float64))
	if taskAssigned.Data["assignee"] != username {
		t.Fatalf("task assignee = %#v", taskAssigned.Data)
	}
	taskComment := runner.jsonOK("task", "comment", taskTarget, "--body", "task integration comment", "--base-version", strconv.Itoa(taskVersion))
	taskVersion = int(taskComment.Data["version"].(float64))
	watchedTask := runner.jsonOK("task", "watch", taskTarget)
	if watchedTask.Data["watching"] != true || watchedTask.Data["verified"] != true {
		t.Fatalf("task watch was not verified: %#v", watchedTask.Data)
	}
	taskHistory := runner.jsonOK("task", "history", taskTarget, "--type", "comment", "--fields", "id,kind,author,comment")
	if !containsComment(taskHistory.Items, "task integration comment") {
		t.Fatalf("task comment missing from CLI history: %#v", taskHistory.Items)
	}
	unwatchedTask := runner.jsonOK("task", "unwatch", taskTarget)
	if unwatchedTask.Data["watching"] != false || unwatchedTask.Data["verified"] != true {
		t.Fatalf("task unwatch was not verified: %#v", unwatchedTask.Data)
	}
	stdout, stderr, code = runner.run("story", "close", strconv.Itoa(storyWithTaskRef), "--status", closedStoryStatus, "--base-version", strconv.Itoa(storyWithTaskVersion))
	if code != 0 || !strings.Contains(stderr, "open tasks keep this story active") {
		t.Fatalf("open-task close warning exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	openTaskStory := runner.jsonOK("story", "view", strconv.Itoa(storyWithTaskRef), "--fields", "status,is_closed")
	if openTaskStory.Data["status"] != closedStoryStatus || openTaskStory.Data["is_closed"] != false {
		t.Fatalf("open-task story close semantics = %#v", openTaskStory.Data)
	}
	closedTaskStatus := firstClosedTaskStatus(t, baseURL, token, projectID)
	doneTask := runner.jsonOK("task", "done", taskTarget, "--status", closedTaskStatus, "--base-version", strconv.Itoa(taskVersion))
	if doneTask.Data["is_closed"] != true {
		t.Fatalf("task not done: %#v", doneTask.Data)
	}
	taskVersion = int(doneTask.Data["version"].(float64))
	completedParent := runner.jsonOK("story", "view", strconv.Itoa(storyWithTaskRef), "--fields", "status,is_closed")
	if completedParent.Data["status"] != closedStoryStatus || completedParent.Data["is_closed"] != true {
		t.Fatalf("parent Story did not close after final task: %#v", completedParent.Data)
	}
	openTaskStatus := firstOpenTaskStatus(t, baseURL, token, projectID)
	reopenedTask := runner.jsonOK("task", "reopen", taskTarget, "--status", openTaskStatus, "--base-version", strconv.Itoa(taskVersion))
	if reopenedTask.Data["is_closed"] != false {
		t.Fatalf("task not reopened: %#v", reopenedTask.Data)
	}
	taskVersion = int(reopenedTask.Data["version"].(float64))
	unassignedTask := runner.jsonOK("task", "unassign", taskTarget, "--base-version", strconv.Itoa(taskVersion))
	if value, ok := unassignedTask.Data["assignee"]; ok && value != "" {
		t.Fatalf("task not unassigned: %#v", unassignedTask.Data)
	}
	taskVersion = int(unassignedTask.Data["version"].(float64))
	standaloneSprintTask := runner.jsonOK("task", "move", taskTarget, "--sprint", milestoneSlug, "--base-version", strconv.Itoa(taskVersion))
	if value, ok := standaloneSprintTask.Data["story_ref"]; ok && value != float64(0) {
		t.Fatalf("task still has parent Story: %#v", standaloneSprintTask.Data)
	}
	if standaloneSprintTask.Data["sprint_slug"] != milestoneSlug {
		t.Fatalf("standalone task missing sprint: %#v", standaloneSprintTask.Data)
	}
	taskVersion = int(standaloneSprintTask.Data["version"].(float64))
	parentedTask := runner.jsonOK("task", "move", taskTarget, "--story", strconv.Itoa(storyWithTaskRef), "--base-version", strconv.Itoa(taskVersion))
	if parentedTask.Data["story_ref"] != float64(storyWithTaskRef) || parentedTask.Data["sprint_slug"] != milestoneSlug {
		t.Fatalf("task did not rejoin parent Story: %#v", parentedTask.Data)
	}
	taskVersion = int(parentedTask.Data["version"].(float64))
	backlogTask := runner.jsonOK("task", "move", taskTarget, "--sprint", "backlog", "--base-version", strconv.Itoa(taskVersion))
	if value, ok := backlogTask.Data["story_ref"]; ok && value != float64(0) {
		t.Fatalf("backlog task still has parent Story: %#v", backlogTask.Data)
	}
	if value, ok := backlogTask.Data["sprint_slug"]; ok && value != "" {
		t.Fatalf("backlog task still has sprint: %#v", backlogTask.Data)
	}

	editedSprint := runner.jsonOK("sprint", "edit", milestoneSlug, "--finish", "2026-09-08")
	if editedSprint.Data["finish"] != "2026-09-08" {
		t.Fatalf("sprint finish was not updated: %#v", editedSprint.Data)
	}
	closedSprint := runner.jsonOK("sprint", "close", milestoneSlug)
	if closedSprint.Data["closed"] != true {
		t.Fatalf("sprint not closed: %#v", closedSprint.Data)
	}
	closedSprints := runner.jsonOK("sprint", "list", "--state", "closed", "--fields", "slug,closed")
	if !containsSlug(closedSprints.Items, milestoneSlug) {
		t.Fatalf("closed sprint %q missing from closed list", milestoneSlug)
	}
	reopenedSprint := runner.jsonOK("sprint", "reopen", milestoneSlug)
	if reopenedSprint.Data["closed"] != false {
		t.Fatalf("sprint not reopened: %#v", reopenedSprint.Data)
	}

	wikiSlug := "e2e-guide"
	wiki := runner.jsonOK("wiki", "create", "--slug", wikiSlug, "--body", "# E2E guide\nInitial content")
	wikiID := int64(wiki.Data["id"].(float64))
	wikiVersion := int(wiki.Data["version"].(float64))
	wikiPages := runner.jsonOK("wiki", "list", "--fields", "slug,version,editions")
	if !containsSlug(wikiPages.Items, wikiSlug) {
		t.Fatalf("created wiki page %q missing from list", wikiSlug)
	}
	wikiView := runner.jsonOK("wiki", "view", wikiSlug, "--fields", "slug,content,version,is_watcher")
	if wikiView.Data["content"] != "# E2E guide\nInitial content" {
		t.Fatalf("wiki content=%#v", wikiView.Data)
	}
	wikiDryRun := runner.jsonOK("wiki", "edit", wikiSlug, "--body", "must not persist", "--dry-run")
	if wikiDryRun.Plan["performed"] != false || wikiDryRun.Plan["would_write"] != true {
		t.Fatalf("wiki dry-run plan=%#v", wikiDryRun.Plan)
	}
	wikiEdited := runner.jsonOKWithInput("# E2E guide\nUpdated from stdin", "wiki", "edit", wikiSlug, "--body-file", "-", "--base-version", strconv.Itoa(wikiVersion))
	wikiVersion = int(wikiEdited.Data["version"].(float64))
	if wikiEdited.Data["content"] != "# E2E guide\nUpdated from stdin" {
		t.Fatalf("wiki stdin update=%#v", wikiEdited.Data)
	}
	var externalWikiUpdate map[string]any
	apiRequest(t, http.MethodPatch, baseURL+"wiki/"+strconv.FormatInt(wikiID, 10), token, map[string]any{"content": "external wiki edit", "version": wikiVersion}, &externalWikiUpdate)
	stdout, stderr, code = runner.run("--json", "wiki", "edit", wikiSlug, "--body", "must conflict", "--base-version", strconv.Itoa(wikiVersion))
	if code != 6 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("stale wiki OCC edit exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	watchedWiki := runner.jsonOK("wiki", "watch", wikiSlug)
	if watchedWiki.Data["watching"] != true || watchedWiki.Data["verified"] != true {
		t.Fatalf("wiki watch was not verified: %#v", watchedWiki.Data)
	}
	wikiHistory := runner.jsonOK("wiki", "history", wikiSlug, "--type", "activity", "--fields", "id,kind,author,changes")
	if len(wikiHistory.Items) == 0 {
		t.Fatalf("wiki activity history is empty")
	}
	unwatchedWiki := runner.jsonOK("wiki", "unwatch", wikiSlug)
	if unwatchedWiki.Data["watching"] != false || unwatchedWiki.Data["verified"] != true {
		t.Fatalf("wiki unwatch was not verified: %#v", unwatchedWiki.Data)
	}
	stdout, stderr, code = runner.run("--json", "--no-input", "wiki", "delete", wikiSlug)
	if code != 10 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("unconfirmed wiki delete exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	deletedWiki := runner.jsonOK("wiki", "delete", wikiSlug, "--yes")
	if deletedWiki.Data["deleted"] != true {
		t.Fatalf("wiki page not deleted: %#v", deletedWiki.Data)
	}
	wikiPages = runner.jsonOK("wiki", "list", "--fields", "slug")
	if containsSlug(wikiPages.Items, wikiSlug) {
		t.Fatalf("deleted wiki page %q remains in list", wikiSlug)
	}

	searchResults := runner.jsonOK("search", "external", "--fields", "kind,ref,subject")
	if !containsKind(searchResults.Items, "issue") || !containsKind(searchResults.Items, "story") {
		t.Fatalf("search did not return issue and story results: %#v", searchResults.Items)
	}
	taskSearch := runner.jsonOK("search", "task", "--type", "task", "--fields", "kind,ref,subject")
	if len(taskSearch.Items) == 0 || !containsKind(taskSearch.Items, "task") {
		t.Fatalf("task search returned no tasks: %#v", taskSearch.Items)
	}

	invalidRunner := runner
	invalidRunner.env = replaceEnv(runner.env, "TAIGA_TOKEN", "token-that-must-never-appear")
	stdout, stderr, code = invalidRunner.run("--verbose", "auth", "status")
	if code != 3 {
		t.Fatalf("invalid token exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "token-that-must-never-appear") {
		t.Fatalf("credential leaked: stdout=%s stderr=%s", stdout, stderr)
	}
}

type cliRunner struct {
	t      *testing.T
	binary string
	dir    string
	env    []string
}

func (r cliRunner) run(args ...string) (string, string, int) {
	return r.runWithInput("", args...)
}

func (r cliRunner) runWithInput(input string, args ...string) (string, string, int) {
	r.t.Helper()
	command := exec.Command(r.binary, args...)
	command.Dir = r.dir
	command.Env = append(os.Environ(), r.env...)
	if input != "" {
		command.Stdin = strings.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		r.t.Fatalf("run %v: %v", args, err)
	}
	return stdout.String(), stderr.String(), exitErr.ExitCode()
}

func (r cliRunner) jsonOKWithInput(input string, args ...string) envelope {
	r.t.Helper()
	args = append([]string{"--json"}, args...)
	stdout, stderr, code := r.runWithInput(input, args...)
	if code != 0 {
		r.t.Fatalf("taiga %v exit=%d stderr=%s", args, code, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		r.t.Fatalf("taiga %v wrote stderr on success: %s", args, stderr)
	}
	var result envelope
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		r.t.Fatalf("taiga %v returned invalid JSON: %v: %s", args, err, stdout)
	}
	return result
}

func (r cliRunner) jsonOK(args ...string) envelope {
	r.t.Helper()
	args = append([]string{"--json"}, args...)
	stdout, stderr, code := r.run(args...)
	if code != 0 {
		r.t.Fatalf("taiga %v exit=%d stderr=%s", args, code, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		r.t.Fatalf("taiga %v wrote stderr on success: %s", args, stderr)
	}
	var result envelope
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		r.t.Fatalf("taiga %v returned invalid JSON: %v: %s", args, err, stdout)
	}
	return result
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func register(t *testing.T, baseURL, username, password string) string {
	t.Helper()
	body := map[string]any{"accepted_terms": true, "email": username + "@localhost.invalid", "full_name": "Taiga CLI E2E", "password": password, "type": "public", "username": username}
	var response map[string]any
	apiRequest(t, http.MethodPost, baseURL+"auth/register", "", body, &response)
	return response["auth_token"].(string)
}

func createProject(t *testing.T, baseURL, token string) map[string]any {
	t.Helper()
	var templates []map[string]any
	apiRequest(t, http.MethodGet, baseURL+"project-templates", token, nil, &templates)
	body := map[string]any{"name": "Taiga CLI E2E", "description": "temporary integration project", "creation_template": templates[0]["id"], "is_private": true}
	var project map[string]any
	apiRequest(t, http.MethodPost, baseURL+"projects", token, body, &project)
	return project
}

func firstClosedStatus(t *testing.T, baseURL, token string, projectID int64) string {
	t.Helper()
	var statuses []map[string]any
	apiRequest(t, http.MethodGet, baseURL+"issue-statuses?project="+strconv.FormatInt(projectID, 10), token, nil, &statuses)
	for _, status := range statuses {
		if status["is_closed"] == true {
			return status["name"].(string)
		}
	}
	t.Fatal("project has no closed issue status")
	return ""
}

func firstClosedStoryStatus(t *testing.T, baseURL, token string, projectID int64) string {
	t.Helper()
	var statuses []map[string]any
	apiRequest(t, http.MethodGet, baseURL+"userstory-statuses?project="+strconv.FormatInt(projectID, 10), token, nil, &statuses)
	for _, status := range statuses {
		if status["is_closed"] == true {
			return status["name"].(string)
		}
	}
	t.Fatal("project has no closed user story status")
	return ""
}

func firstClosedTaskStatus(t *testing.T, baseURL, token string, projectID int64) string {
	t.Helper()
	var statuses []map[string]any
	apiRequest(t, http.MethodGet, baseURL+"task-statuses?project="+strconv.FormatInt(projectID, 10), token, nil, &statuses)
	for _, status := range statuses {
		if status["is_closed"] == true {
			return status["name"].(string)
		}
	}
	t.Fatal("project has no closed task status")
	return ""
}

func firstOpenTaskStatus(t *testing.T, baseURL, token string, projectID int64) string {
	t.Helper()
	var statuses []map[string]any
	apiRequest(t, http.MethodGet, baseURL+"task-statuses?project="+strconv.FormatInt(projectID, 10), token, nil, &statuses)
	for _, status := range statuses {
		if status["is_closed"] == false {
			return status["name"].(string)
		}
	}
	t.Fatal("project has no open task status")
	return ""
}

func apiRequest(t *testing.T, method, url, token string, body, output any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("%s %s returned %d: %s", method, url, response.StatusCode, data)
	}
	if output != nil && len(data) > 0 {
		if err := json.Unmarshal(data, output); err != nil {
			t.Fatalf("decode %s %s: %v: %s", method, url, err, data)
		}
	}
}

func containsRef(items []map[string]any, ref int) bool {
	for _, item := range items {
		if int(item["ref"].(float64)) == ref {
			return true
		}
	}
	return false
}

func containsSlug(items []map[string]any, slug string) bool {
	for _, item := range items {
		if item["slug"] == slug {
			return true
		}
	}
	return false
}

func containsKind(items []map[string]any, kind string) bool {
	for _, item := range items {
		if item["kind"] == kind {
			return true
		}
	}
	return false
}

func containsComment(items []map[string]any, comment string) bool {
	for _, item := range items {
		if item["comment"] == comment {
			return true
		}
	}
	return false
}

func containsID(items []map[string]any, id int64) bool {
	for _, item := range items {
		if int64(item["id"].(float64)) == id {
			return true
		}
	}
	return false
}

func replaceEnv(values []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(values)+1)
	for _, current := range values {
		if !strings.HasPrefix(current, prefix) {
			result = append(result, current)
		}
	}
	return append(result, prefix+value)
}
