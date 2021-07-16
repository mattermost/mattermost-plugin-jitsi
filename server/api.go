package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const externalAPICacheTTL = 3600000

var externalAPICache []byte
var externalAPILastUpdate int64
var externalAPICacheMutex sync.Mutex

type StartMeetingRequest struct {
	ChannelID string `json:"channel_id"`
	Personal  bool   `json:"personal"`
	Topic     string `json:"topic"`
	MeetingID int    `json:"meeting_id"`
}

type StartMeetingFromAction struct {
	model.PostActionIntegrationRequest
	Context struct {
		MeetingID    string `json:"meeting_id"`
		MeetingTopic string `json:"meeting_topic"`
		Personal     bool   `json:"personal"`
		RootID       string `json:"root_id"`
	} `json:"context"`
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch path := r.URL.Path; path {
	case "/api/v1/meetings/enrich":
		p.handleEnrichMeetingJwt(w, r)
	case "/api/v1/meetings":
		p.handleStartMeeting(w, r)
	case "/api/v1/config":
		p.handleConfig(w, r)
	case "/jitsi_meet_external_api.js":
		p.handleExternalAPIjs(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleConfig(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")

	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	config, err := p.getUserConfig(userID)
	if err != nil {
		mlog.Error("Error getting user config", mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		mlog.Error("Error marshaling the Config to json", mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "handleConfig"), mlog.Err(err))
	}
}

func (p *Plugin) handleExternalAPIjs(w http.ResponseWriter, r *http.Request) {
	if p.getConfiguration().JitsiCompatibilityMode {
		p.proxyExternalAPIjs(w, r)
		return
	}

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		mlog.Error("Filed to get the bundle path")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	externalAPIPath := filepath.Join(bundlePath, "assets", "external_api.js")
	externalAPIFile, err := os.Open(externalAPIPath)
	if err != nil {
		mlog.Error("Error opening file", mlog.String("path", externalAPIPath), mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	code, err := ioutil.ReadAll(externalAPIFile)
	if err != nil {
		mlog.Error("Error reading file content", mlog.String("path", externalAPIPath), mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	_, err = w.Write(code)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "proxyExternalAPIjs"), mlog.Err(err))
	}
}

func (p *Plugin) proxyExternalAPIjs(w http.ResponseWriter, r *http.Request) {
	externalAPICacheMutex.Lock()
	defer externalAPICacheMutex.Unlock()

	if externalAPICache != nil && externalAPILastUpdate > (model.GetMillis()-externalAPICacheTTL) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write(externalAPICache)
		return
	}
	resp, err := http.Get(p.getConfiguration().GetJitsiURL() + "/external_api.js")
	if err != nil {
		mlog.Error("Error getting the external_api.js file from your Jitsi instance, please verify your JitsiURL setting", mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		mlog.Error("Error getting reading the content", mlog.String("url", p.getConfiguration().GetJitsiURL()+"/external_api.js"), mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	externalAPICache = body
	externalAPILastUpdate = model.GetMillis()
	w.Header().Set("Content-Type", "application/javascript")
	_, err = w.Write(body)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "proxyExternalAPIjs"), mlog.Err(err))
	}
}

func (p *Plugin) handleStartMeeting(w http.ResponseWriter, r *http.Request) {
	if err := p.getConfiguration().IsValid(); err != nil {
		mlog.Error("Invalid plugin configuration", mlog.Err(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := r.Header.Get("Mattermost-User-Id")

	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		mlog.Debug("Unable to the user", mlog.Err(appErr))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req StartMeetingRequest
	var action StartMeetingFromAction

	bodyData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		mlog.Debug("Unable to read request body", mlog.Err(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err1 := json.NewDecoder(bytes.NewReader(bodyData)).Decode(&req)
	err2 := json.NewDecoder(bytes.NewReader(bodyData)).Decode(&action)
	if err1 != nil && err2 != nil {
		mlog.Debug("Unable to decode the request content as start meeting request or start meeting action")
		http.Error(w, "Unable to decode your request", http.StatusBadRequest)
		return
	}

	channelID := req.ChannelID
	if channelID == "" {
		channelID = action.ChannelId
	}

	if _, err := p.API.GetChannelMember(channelID, userID); err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	channel, appErr := p.API.GetChannel(channelID)
	if appErr != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userConfig, err := p.getUserConfig(userID)
	if err != nil {
		mlog.Error("Error getting user config", mlog.Err(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if userConfig.NamingScheme == jitsiNameSchemeAsk && action.PostId == "" {
		err = p.askMeetingType(user, channel, "")
		if err != nil {
			mlog.Error("Error asking the user for meeting name type", mlog.Err(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte("OK"))
		if err != nil {
			mlog.Warn("Unable to write response body", mlog.String("handler", "handleStartMeeting"), mlog.Err(err))
		}
		return
	}

	var meetingID string
	if userConfig.NamingScheme == jitsiNameSchemeAsk && action.PostId != "" {
		meetingID, err = p.startMeeting(user, channel, action.Context.MeetingID, action.Context.MeetingTopic, action.Context.Personal, "")
		if err != nil {
			mlog.Error("Error starting a new meeting from ask response", mlog.Err(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p.deleteEphemeralPost(action.UserId, action.PostId)
	} else {
		meetingID, err = p.startMeeting(user, channel, "", req.Topic, req.Personal, action.Context.RootID)
		if err != nil {
			mlog.Error("Error starting a new meeting", mlog.Err(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	b, err := json.Marshal(map[string]string{"meeting_id": meetingID})
	if err != nil {
		mlog.Error("Error marshaling the MeetingID to json", mlog.Err(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "handleStartMeeting"), mlog.Err(err))
	}
}

func (p *Plugin) handleEnrichMeetingJwt(w http.ResponseWriter, r *http.Request) {
	if err := p.getConfiguration().IsValid(); err != nil {
		mlog.Error("Invalid plugin configuration", mlog.Err(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := r.Header.Get("Mattermost-User-Id")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var req EnrichMeetingJwtRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Debug("Unable to read request body", mlog.Err(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	var user *model.User
	var err *model.AppError
	user, err = p.API.GetUser(userID)
	if err != nil {
		http.Error(w, err.Error(), err.StatusCode)
	}

	JWTMeeting := p.getConfiguration().JitsiJWT

	if !JWTMeeting {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	meetingJWT, err2 := p.updateJwtUserInfo(req.Jwt, user)
	if err2 != nil {
		mlog.Error("Error updating JWT context", mlog.Err(err2))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	b, err2 := json.Marshal(map[string]interface{}{"jwt": meetingJWT})
	if err2 != nil {
		mlog.Error("Error marshaling the JWT json", mlog.Err(err2))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err2 = w.Write(b)
	if err2 != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "handleEnrichMeetingJwt"), mlog.Err(err))
	}
}
