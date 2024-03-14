package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cristalhq/jwt/v3"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi/i18n"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVerifyJwt(t *testing.T) {
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.mpHl842O7xEZjgQ8CyX8xYLDoEORGVMnAxULkW-u8Ek"
	t.Run("bad secret should fail", func(t *testing.T) {
		claims, err := verifyJwt("bad-secret", testToken)
		require.NotNil(t, err)
		require.Nil(t, claims)
	})
	t.Run("right secret should work", func(t *testing.T) {
		claims, err := verifyJwt("test-secret", testToken)
		require.Nil(t, err)
		require.Equal(t, "1234567890", claims.Subject)
		issuedAt, err := claims.IssuedAt.MarshalJSON()
		require.Nil(t, err)
		require.Equal(t, "1516239022", string(issuedAt))
	})
}

func TestSignClaims(t *testing.T) {
	claims := Claims{}
	claims.Issuer = "test"
	claims.Audience = []string{"test-app-id"}
	claims.ExpiresAt = jwt.NewNumericDate(time.Time{})
	claims.Subject = "test"
	claims.Room = "test"

	token, err := signClaims("test-secret", &claims)
	require.Nil(t, err)

	newClaims, err := verifyJwt("test-secret", token)
	require.Nil(t, err)
	require.Equal(t, &claims, newClaims)
}

func TestStartMeeting(t *testing.T) {
	p := Plugin{
		configuration: &configuration{
			JitsiURL: "http://test",
		},
	}
	apiMock := plugintest.API{}
	defer apiMock.AssertExpectations(t)
	apiMock.On("GetBundlePath").Return("..", nil)
	config := model.Config{}
	config.SetDefaults()
	apiMock.On("GetConfig").Return(&config, nil)

	p.SetAPI(&apiMock)

	i18nBundle, err := i18n.InitBundle(p.API, filepath.Join("assets", "i18n"))
	require.Nil(t, err)
	p.b = i18nBundle

	testUser := model.User{Id: "test-id", Username: "test-username", FirstName: "test-first-name", LastName: "test-last-name", Nickname: "test-nickname"}
	testChannel := model.Channel{Id: "test-id", Type: model.ChannelTypeDirect, Name: "test-name", DisplayName: "test-display-name"}

	t.Run("start meeting without topic or id", func(t *testing.T) {
		apiMock.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return strings.HasPrefix(post.Props["meeting_link"].(string), "http://test/")
		})).Return(&model.Post{}, nil)
		b, _ := json.Marshal(UserConfig{Embedded: false, NamingScheme: "mattermost"})
		apiMock.On("KVGet", "config_test-id", mock.Anything).Return(b, nil)

		meetingID, err := p.startMeeting(&testUser, &testChannel, "", "", "")
		require.Nil(t, err)
		require.Regexp(t, "^test-username-", meetingID)
	})

	t.Run("start meeting with topic and without id", func(t *testing.T) {
		apiMock.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return strings.HasPrefix(post.Props["meeting_link"].(string), "http://test/")
		})).Return(&model.Post{}, nil)
		b, _ := json.Marshal(UserConfig{Embedded: false, NamingScheme: "mattermost"})
		apiMock.On("KVGet", "config_test-id", mock.Anything).Return(b, nil)

		meetingID, err := p.startMeeting(&testUser, &testChannel, "", "Test topic", "")
		require.Nil(t, err)
		require.Regexp(t, "^Test-topic-", meetingID)
	})

	t.Run("start meeting without topic and with id", func(t *testing.T) {
		apiMock.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return strings.HasPrefix(post.Props["meeting_link"].(string), "http://test/")
		})).Return(&model.Post{}, nil)
		b, _ := json.Marshal(UserConfig{Embedded: false, NamingScheme: "mattermost"})
		apiMock.On("KVGet", "config_test-id", mock.Anything).Return(b, nil)

		meetingID, err := p.startMeeting(&testUser, &testChannel, "test-id", "", "")
		require.Nil(t, err)
		require.Regexp(t, "^test-username-", meetingID)
	})

	t.Run("start meeting with topic and id", func(t *testing.T) {
		testUser := model.User{Id: "test-id", Username: "test-username", FirstName: "test-first-name", LastName: "test-last-name", Nickname: "test-nickname"}
		testChannel := model.Channel{Id: "test-id", Type: model.ChannelTypeOpen, TeamId: "test-team-id", Name: "test-name", DisplayName: "test-display-name"}

		apiMock.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return strings.HasPrefix(post.Props["meeting_link"].(string), "http://test/")
		})).Return(&model.Post{}, nil)
		b, _ := json.Marshal(UserConfig{Embedded: false, NamingScheme: "mattermost"})
		apiMock.On("KVGet", "config_test-id", mock.Anything).Return(b, nil)

		meetingID, err := p.startMeeting(&testUser, &testChannel, "test-id", "Test topic", "")
		require.Nil(t, err)
		require.Equal(t, "test-id", meetingID)
	})
}
