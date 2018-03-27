package plugin

import (
	"encoding/json"
	"strings"

	"github.com/arquillian/ike-prow-plugins/pkg/github"
	"github.com/arquillian/ike-prow-plugins/pkg/log"
	"github.com/arquillian/ike-prow-plugins/pkg/scm"
	"github.com/arquillian/ike-prow-plugins/pkg/utils"
	gogh "github.com/google/go-github/github"
)

// ProwPluginName is an external prow plugin name used to register this service
const ProwPluginName = "work-in-progress"

// GitHubWIPPRHandler handles PR events and updates status of the PR based on work-in-progress indicator
type GitHubWIPPRHandler struct {
	Client *gogh.Client
}

var (
	handledPrActions = []string{"opened", "reopened", "edited"}
)

// HandleEvent is an entry point for the plugin logic. This method is invoked by the Server when
// events are dispatched from the /hook service
func (gh *GitHubWIPPRHandler) HandleEvent(log log.Logger, eventType github.EventType, payload []byte) error {
	switch eventType {
	case github.PullRequest:
		var event gogh.PullRequestEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Errorf("Failed while parsing '%q' event with payload: %q. Cause: %q", github.PullRequest, event, err)
			return err
		}

		if !utils.Contains(handledPrActions, *event.Action) {
			return nil
		}

		change := scm.RepositoryChange{
			Owner:    *event.Repo.Owner.Login,
			RepoName: *event.Repo.Name,
			Hash:     *event.PullRequest.Head.SHA,
		}
		statusContext := github.StatusContext{BotName: "ike-plugins", PluginName: ProwPluginName}
		statusService := github.NewStatusService(gh.Client, log, change, statusContext)
		if gh.IsWorkInProgress(event.PullRequest.Title) {
			return statusService.Failure("PR is in progress and can't be merged yet. You might want to wait with review as well")
		}
		return statusService.Success("PR is ready for review and merge")

	default:
		log.Warnf("received an event of type %q but didn't ask for it", eventType)
	}

	return nil
}

// IsWorkInProgress checks if title is marked as Work In Progress
func (gh *GitHubWIPPRHandler) IsWorkInProgress(title *string) bool {
	return strings.HasPrefix(strings.ToLower(*title), "wip ")
}