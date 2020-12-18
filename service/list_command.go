package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func listAsBlock(resp *ListResponse) (blocks []slack.Block) {
	if len(resp.Users) == 0 {
		blocks = make([]slack.Block, 1)
		empty := slack.NewTextBlockObject("mrkdwn", "No users in queue.", false, false)
		blocks[0] = slack.NewSectionBlock(empty, nil, nil)
		return
	}
	blocks = make([]slack.Block, len(resp.Users)*3)
	for i, user := range resp.Users {
		blocks[i*3] = slack.NewDividerBlock()
		userinfo := fmt.Sprintf("*%d:* %s\n*Wait time:* %s\n*Topic:* %s", i+1, userToLink(user), (time.Now().Sub(resp.Times[i])).String(), resp.Metadata[i])
		userblock := slack.NewTextBlockObject("mrkdwn", userinfo, false, false)
		iconblock := slack.NewImageBlockElement(user.Profile.Image24, user.Name)
		blocks[i*3+1] = slack.NewSectionBlock(userblock, nil, slack.NewAccessory(iconblock))
		remove := slack.NewButtonBlockElement("remove", GenerateActionValue(i, resp.Token), slack.NewTextBlockObject("plain_text", "Remove", false, false))
		take := slack.NewButtonBlockElement("take", GenerateActionValue(i, resp.Token), slack.NewTextBlockObject("plain_text", "Dequeue", false, false))
		blocks[i*3+2] = slack.NewActionBlock(fmt.Sprintf("actions_%v", user.ID), remove, take)
	}

	return
}

func updateListInUI(action *slack.InteractionCallback, s *QueueService, api *slack.Client) {
	lreq := &ListRequest{}
	lresp := &ListResponse{}
	err := s.List(lreq, lresp)
	if err != nil {
		glog.Errorf("Error getting queue state: %v", err)
		return
	}
	_, _, err = api.PostMessage("",
		slack.MsgOptionResponseURL(action.ResponseURL, slack.ResponseTypeEphemeral),
		slack.MsgOptionReplaceOriginal(action.ResponseURL),
		slack.MsgOptionBlocks(listAsBlock(lresp)...))
	if err != nil {
		glog.Errorf("Error posting reply: %v")
	}
}

func (c *ListCommand) Handle(cmd *slack.SlashCommand, s *QueueService, w http.ResponseWriter) (err error) {
	// Check permission to list queue.
	user := &slack.User{ID: cmd.UserID}
	ok, err := c.perms.IsAdmin(user)
	if err != nil {
		glog.Errorf("Error checking admin status of %v (%v): %v", cmd.UserID, cmd.UserName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		glog.Errorf("Permission denied to user %v (%v)", cmd.UserID, cmd.UserName)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := ListRequest{}
	resp := ListResponse{}
	err = s.List(&req, &resp)
	if err != nil {
		glog.Errorf("Error listing users: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	blocks := listAsBlock(&resp)
	msg := slack.NewBlockMessage(blocks...)
	b, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		glog.Fatalf("Error marshalling json: %v", err)
	}
	glog.V(2).Infof("List response:\n%v", string(b))
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return
}
