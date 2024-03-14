package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const externalAPICacheTTL = 3600000
const mattermostUserIDHeader = "Mattermost-User-Id"

var externalAPICache []byte
var externalAPILastUpdate int64
var externalAPICacheMutex sync.Mutex

type StartMeetingRequest struct {
	ChannelID string `json:"channel_id"`
	Topic     string `json:"topic"`
	Personal  bool   `json:"personal"`
	MeetingID int    `json:"meeting_id"`
}

type StartMeetingFromAction struct {
	model.PostActionIntegrationRequest
	Context struct {
		MeetingID    string `json:"meeting_id"`
		MeetingTopic string `json:"meeting_topic"`
		RootID       string `json:"root_id"`
		Personal     bool   `json:"personal"`
	} `json:"context"`
}

type JaaSSettingsFromAction struct {
	Jwt  string `json:"jaasJwt"`
	Path string `json:"jaasPath"`
}

func (p *Plugin) InitAPI() *mux.Router {
	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api/v1").Subrouter()

	r.HandleFunc("/jitsi_meet_external_api.js", p.handleExternalAPIjs)
	apiRouter.HandleFunc("/meetings/enrich", p.checkAuth(p.handleEnrichMeetingJwt)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/meetings", p.checkAuth(p.handleStartMeeting)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/config", p.checkAuth(p.handleConfig)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/meetings/jaas/settings", p.handleJaaSSettings)
	r.HandleFunc("{anything:.*}", http.NotFound)

	return r
}

func (p *Plugin) checkAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mattermostUserID := r.Header.Get(mattermostUserIDHeader)
		if mattermostUserID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

func (p *Plugin) handleJaaSSettings(w http.ResponseWriter, r *http.Request) {
	if !p.getConfiguration().UseJaaS {
		errString := "Error JaaS requested while disabled"
		mlog.Error(errString)
		http.Error(w, errString, http.StatusBadRequest)
		return
	}

	if err := p.getConfiguration().IsValid(); err != nil {
		mlog.Error("Invalid plugin configuration", mlog.Err(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var jaasSettingsFromAction JaaSSettingsFromAction
	if err := json.NewDecoder(r.Body).Decode(&jaasSettingsFromAction); err != nil {
		mlog.Debug("Unable to decode the request content as start meeting request or start meeting action")
		http.Error(w, "Unable to decode your request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var user *model.User
	userID := r.Header.Get(mattermostUserIDHeader)
	if userID != "" {
		// Handle moderator
		userResult, appErr := p.API.GetUser(userID)
		if appErr != nil {
			mlog.Debug("Unable to get the user", mlog.Err(appErr))
			http.Error(w, "You are forbidden to get the user", http.StatusForbidden)
			return
		}
		user = userResult
	}

	jaasSettings, err := p.getJaaSSettings(jaasSettingsFromAction.Jwt, jaasSettingsFromAction.Path, user)
	if err != nil {
		mlog.Error("Error getting JaaSSettings", mlog.Err(err))
		http.Error(w, "Invalid JaaS settings", http.StatusBadRequest)
		return
	}

	settingsJSON, err := json.Marshal(jaasSettings)
	if err != nil {
		mlog.Error("Error marshaling the JaaSSettings to json", mlog.Err(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(settingsJSON)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "handleJaaSSettings"), mlog.Err(err))
	}
}

func (p *Plugin) handleConfig(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get(mattermostUserIDHeader)

	config, err := p.getUserConfig(userID)
	if err != nil {
		mlog.Error("Error getting user config", mlog.Err(err))
		http.Error(w, "Error getting user config", http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		mlog.Error("Error marshaling the Config to json", mlog.Err(err))
		http.Error(w, "Error marshaling the Config to json", http.StatusInternalServerError)
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
		if p.getConfiguration().UseJaaS {
			p.proxyExternalAPIjsJaaS(w, r)
			return
		}
		p.proxyExternalAPIjs(w, r)
		return
	}

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		mlog.Error("Failed to get the bundle path")
		http.Error(w, "Failed to get the bundle path", http.StatusInternalServerError)
		return
	}
	externalAPIPath := filepath.Join(bundlePath, "assets", "external_api.js")
	externalAPIFile, err := os.Open(externalAPIPath)
	if err != nil {
		mlog.Error("Error opening file", mlog.String("path", externalAPIPath), mlog.Err(err))
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	code, err := io.ReadAll(externalAPIFile)
	if err != nil {
		mlog.Error("Error reading file content", mlog.String("path", externalAPIPath), mlog.Err(err))
		http.Error(w, "Error reading file content", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	_, err = w.Write(code)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "proxyExternalAPIjs"), mlog.Err(err))
	}
}

func (p *Plugin) proxyExternalAPIjsJaaS(w http.ResponseWriter, _ *http.Request) {
	externalAPICacheMutex.Lock()
	defer externalAPICacheMutex.Unlock()

	if externalAPICache != nil && externalAPILastUpdate > (model.GetMillis()-externalAPICacheTTL) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = w.Write(externalAPICache)
		return
	}
	resp, err := http.Get(fmt.Sprintf("%s/libs/external_api.min.js", p.getConfiguration().Get8x8vcURL()))
	if err != nil {
		mlog.Error("Error getting the external_api.min.js file from your 8x8", mlog.Err(err))
		http.Error(w, "Error getting the external_api.min.js file from your 8x8", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mlog.Error("Error reading the content", mlog.String("url", fmt.Sprintf("%s/external_api.min.js", p.getConfiguration().Get8x8vcURL())), mlog.Err(err))
		http.Error(w, "Error reading the content", http.StatusInternalServerError)
		return
	}
	externalAPICache = body
	externalAPILastUpdate = model.GetMillis()
	w.Header().Set("Content-Type", "application/javascript")
	_, err = w.Write(body)
	if err != nil {
		mlog.Warn("Unable to write response body", mlog.String("handler", "proxyExternalAPIjsJaaS"), mlog.Err(err))
	}
}

func (p *Plugin) proxyExternalAPIjs(w http.ResponseWriter, _ *http.Request) {
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
		http.Error(w, "Error getting the external_api.js file from your Jitsi instance, please verify your JitsiURL setting", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mlog.Error("Error reading external_api.js file", mlog.String("url", p.getConfiguration().GetJitsiURL()+"/external_api.js"), mlog.Err(err))
		http.Error(w, "Error reading external_api.js file", http.StatusInternalServerError)
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

	userID := r.Header.Get(mattermostUserIDHeader)

	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		mlog.Debug("Unable to get the user", mlog.Err(appErr))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req StartMeetingRequest
	var action StartMeetingFromAction

	bodyData, err := io.ReadAll(r.Body)
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
		meetingID, err = p.startMeeting(user, channel, action.Context.MeetingID, action.Context.MeetingTopic, "")
		if err != nil {
			mlog.Error("Error starting a new meeting from ask response", mlog.Err(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p.deleteEphemeralPost(action.UserId, action.PostId)
	} else {
		meetingID, err = p.startMeeting(user, channel, "", req.Topic, action.Context.RootID)
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

	userID := r.Header.Get(mattermostUserIDHeader)

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

	JWTMeeting := p.getConfiguration().JitsiJWT || p.getConfiguration().UseJaaS

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
