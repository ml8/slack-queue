package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func listAsBlock(resp *ListResponse) (b []byte) {
	blocks := make([]slack.Block, len(resp.Users)*3)
	for i, user := range resp.Users {
		blocks[i*3] = slack.NewDividerBlock()
		userinfo := fmt.Sprintf("*%d:* %s\nwait time: %s", i+1, userToLink(user), (time.Now().Sub(resp.Times[i])).String())
		userblock := slack.NewTextBlockObject("mrkdwn", userinfo, false, false)
		iconblock := slack.NewImageBlockElement(user.Profile.Image24, user.Name)
		blocks[i*3+1] = slack.NewSectionBlock(userblock, nil, slack.NewAccessory(iconblock))
		remove := slack.NewButtonBlockElement("remove", GenerateActionValue(i, resp.Token), slack.NewTextBlockObject("plain_text", "Remove", false, false))
		blocks[i*3+2] = slack.NewActionBlock(fmt.Sprintf("actions_%v", user.ID), remove)
	}

	msg := slack.NewBlockMessage(blocks...)
	b, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		glog.Fatalf("Error marshalling json: %v", err)
	}
	glog.V(2).Infof("List response:\n%v", string(b))
	return
}

func (c *ListCommand) Handle(cmd *slack.SlashCommand, s *Service, w http.ResponseWriter) (err error) {
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
	b := listAsBlock(&resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return
}
