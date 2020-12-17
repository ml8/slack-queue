package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"time"
)

type AdminInterface interface {
	IsAdmin(user *slack.User) (ok bool, err error)
	SendAdminMessage(str string) (err error)
}

const maxChannelCacheAge = "1h"
const maxRetries = 10

type AdminInterfaceImpl struct {
	adminChan       string
	api             *slack.Client
	chanId          string
	stale           bool
	users           []string
	lastRefreshTime time.Time
	retries         int
}

func MakeChannelPermissionChecker(api *slack.Client, adminChan string) AdminInterface {
	return &AdminInterfaceImpl{api: api, adminChan: adminChan, stale: true}
}

// TODO refactor into generic function to handle paginated functions (doesn't
// this exist in API?
func getChannels(api *slack.Client) (chans []slack.Channel, err error) {
	types := []string{"public_channel", "private_channel"}
	params := slack.GetConversationsParameters{Types: types}
	for {
		c, nc, e := api.GetConversations(&params)
		params.Cursor = nc
		if e != nil {
			err = e
			glog.Errorf("Error retrieving channels: %v", err)
			return
		}
		glog.V(2).Infof("Got %d channels", len(c))
		chans = append(chans, c...)
		if nc == "" {
			// Done when cursor is empty
			break
		}
	}
	return
}

func getUsersInChannel(api *slack.Client, id string) (users []string, err error) {
	params := slack.GetUsersInConversationParameters{ChannelID: id}
	for {
		u, nc, e := api.GetUsersInConversation(&params)
		params.Cursor = nc
		if e != nil {
			err = e
			glog.Errorf("Error retrieving users in channel: %v", err)
			return
		}
		glog.V(2).Info("Got %d users", len(u))
		users = append(users, u...)
		if nc == "" {
			break
		}
	}
	return
}

func (p *AdminInterfaceImpl) maybeRefresh() (err error) {
	if p.retries > maxRetries {
		glog.Fatalf("Could not retrieve admin users; failing.")
	}
	maxAge, _ := time.ParseDuration(maxChannelCacheAge)
	age := time.Now().Sub(p.lastRefreshTime)
	stale := p.stale || age > maxAge
	if !stale {
		glog.V(1).Infof("Not refreshing... refreshed at %v", age)
		return
	}

	channels, err := getChannels(p.api)
	if err != nil {
		p.retries++
		glog.Errorf("Error retrieving channels: %v", err)
		return
	}
	glog.V(2).Infof("Got %d channels", len(channels))
	for _, channel := range channels {
		glog.V(2).Infof("Channel: %v", channel.Name)
		if channel.Name == p.adminChan {
			p.chanId = channel.ID
			p.users, err = getUsersInChannel(p.api, channel.ID)
			if err != nil {
				glog.Errorf("Error retrieving users in channel %v: %v", channel.Name, err)
				p.retries++
				return
			}
			p.stale = false
			p.lastRefreshTime = time.Now()
			p.retries = 0
			return
		}
	}
	glog.Errorf("Could not find admin channel. Retrying.")
	p.retries++
	return
}

func (p *AdminInterfaceImpl) IsAdmin(user *slack.User) (ok bool, err error) {
	ok = false
	err = p.maybeRefresh()
	if err != nil {
		return
	}
	glog.V(2).Infof("Channel members: %v", p.users)
	for _, id := range p.users {
		glog.V(2).Infof("Member: %v, User: %v", id, user.ID)
		if id == user.ID {
			ok = true
			return
		}
	}
	return
}

func (p *AdminInterfaceImpl) SendAdminMessage(msg string) (err error) {
	err = p.maybeRefresh()
	if err != nil {
		return
	}
	_, _, err = p.api.PostMessage(p.chanId,
		slack.MsgOptionText(msg, false),
		slack.MsgOptionAsUser(true))
	return
}
