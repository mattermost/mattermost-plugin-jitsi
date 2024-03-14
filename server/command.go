package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/command"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pkg/errors"
)

const jitsiCommand = "jitsi"

const jitsiSettingsSeeCommand = "see"
const jitsiStartCommand = "start"

const valueTrue = "true"
const valueFalse = "false"

const commandArgShowPrejoinPage = "show_prejoin_page"
const commandArgEmbedded = "embedded"
const commandArgNamingScheme = "naming_scheme"

func startMeetingError(channelID string, detailedError string) (*model.CommandResponse, *model.AppError) {
	return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			ChannelId:    channelID,
			Text:         "We could not start a meeting at this time.",
		}, &model.AppError{
			Message:       "We could not start a meeting at this time.",
			DetailedError: detailedError,
		}
}

func (p *Plugin) createJitsiCommand() (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}
	return &model.Command{
		Trigger:              jitsiCommand,
		AutoComplete:         true,
		AutoCompleteDesc:     "Start a Jitsi meeting in current channel. Other available commands: start, help, settings",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(),
		AutocompleteIconData: iconData,
	}, nil
}

func getAutocompleteData() *model.AutocompleteData {
	jitsi := model.NewAutocompleteData("jitsi", "[command]", "Start a Jitsi meeting in current channel. Other available commands: start, help, settings")

	start := model.NewAutocompleteData(jitsiStartCommand, "[topic]", "Start a new meeting in the current channel")
	start.AddTextArgument("(optional) The topic of the new meeting", "[topic]", "")
	jitsi.AddCommand(start)

	help := model.NewAutocompleteData("help", "", "Get slash command help")
	jitsi.AddCommand(help)

	settings := model.NewAutocompleteData("settings", "[setting] [value]", "Update your user settings (see /jitsi help for available options)")

	see := model.NewAutocompleteData(jitsiSettingsSeeCommand, "", "See your current settings")
	settings.AddCommand(see)

	embedded := model.NewAutocompleteData(commandArgEmbedded, "[value]", "Choose where the Jitsi meeting should open")
	items := []model.AutocompleteListItem{{
		HelpText: "Jitsi meeting is embedded as a floating window inside Mattermost",
		Item:     valueTrue,
	}, {
		HelpText: "Jitsi meeting opens in a new window",
		Item:     valueFalse,
	}}
	embedded.AddStaticListArgument("Choose where the Jitsi meeting should open", true, items)
	settings.AddCommand(embedded)

	showPrejoinPage := model.NewAutocompleteData(commandArgShowPrejoinPage, "[value]", "Choose whether the pre-join page should be visible on Jitsi meeting in embedded mode")
	items = []model.AutocompleteListItem{{
		HelpText: "Pre-join page on Jitsi meeting will be displayed",
		Item:     valueTrue,
	}, {
		HelpText: "Pre-join page on Jitsi meeting will not be displayed",
		Item:     valueFalse,
	}}
	showPrejoinPage.AddStaticListArgument("Choose whether the pre-join page should be visible on Jitsi meeting in embedded mode", true, items)
	settings.AddCommand(showPrejoinPage)

	namingScheme := model.NewAutocompleteData(commandArgNamingScheme, "[value]", "Select how meeting names are generated")
	items = []model.AutocompleteListItem{{
		HelpText: "Random English words in title case (e.g. PlayfulDragonsObserveCuriously)",
		Item:     "words",
	}, {
		HelpText: "UUID (universally unique identifier)",
		Item:     "uuid",
	}, {
		HelpText: "Mattermost specific names. Combination of team name, channel name and random text in public and private channels; personal meeting name in direct and group messages channels",
		Item:     "mattermost",
	}, {
		HelpText: "The plugin asks you to select the name every time you start a meeting",
		Item:     "ask",
	}}
	namingScheme.AddStaticListArgument("Choose where the Jitsi meeting should open", true, items)
	settings.AddCommand(namingScheme)
	jitsi.AddCommand(settings)

	return jitsi
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	var parameters []string
	action := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if command != "/"+jitsiCommand {
		return &model.CommandResponse{}, nil
	}

	switch action {
	case "help":
		return p.executeHelpCommand(c, args)

	case "settings":
		return p.executeSettingsCommand(c, args, parameters)

	case jitsiStartCommand:
		fallthrough
	default:
		return p.executeStartMeetingCommand(c, args)
	}
}

func (p *Plugin) executeStartMeetingCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := strings.TrimSpace(strings.TrimPrefix(args.Command, "/"+jitsiCommand))
	input = strings.TrimSpace(strings.TrimPrefix(input, jitsiStartCommand))

	user, appErr := p.API.GetUser(args.UserId)
	if appErr != nil {
		return startMeetingError(args.ChannelId, fmt.Sprintf("getUser() threw error: %s", appErr))
	}

	channel, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		return startMeetingError(args.ChannelId, fmt.Sprintf("getChannel() threw error: %s", appErr))
	}

	userConfig, err := p.getUserConfig(args.UserId)
	if err != nil {
		return startMeetingError(args.ChannelId, fmt.Sprintf("getChannel() threw error: %s", err))
	}

	if userConfig.NamingScheme == jitsiNameSchemeAsk && input == "" {
		if err := p.askMeetingType(user, channel, args.RootId); err != nil {
			return startMeetingError(args.ChannelId, fmt.Sprintf("startMeeting() threw error: %s", appErr))
		}
	} else {
		if _, err := p.startMeeting(user, channel, "", input, args.RootId); err != nil {
			return startMeetingError(args.ChannelId, fmt.Sprintf("startMeeting() threw error: %s", appErr))
		}
	}

	p.trackMeeting(args)

	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeHelpCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	l := p.b.GetUserLocalizer(args.UserId)
	helpTitle := p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: "jitsi.command.help.title",
			Other: `###### Mattermost Jitsi Plugin - Slash Command help
`,
		},
	})
	commandHelp := p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: "jitsi.command.help.text",
			Other: `* |/jitsi| - Create a new meeting
* |/jitsi start [topic]| - Create a new meeting with specified topic
* |/jitsi help| - Show this help text
* |/jitsi settings see| - View your current user settings for the Jitsi plugin
* |/jitsi settings [setting] [value]| - Update your user settings (see below for options)

###### Jitsi Settings:
* |/jitsi settings embedded [true/false]|: (Experimental) When true, Jitsi meeting is embedded as a floating window inside Mattermost. When false, Jitsi meeting opens in a new window.
* |/jitsi settings show_prejoin_page [true/false]|: When false, pre-join page will not be displayed when Jitsi meet is embedded inside Mattermost.
* |/jitsi settings naming_scheme [words/uuid/mattermost/ask]|: Select how meeting names are generated with one of these options:
    * |words|: Random English words in title case (e.g. PlayfulDragonsObserveCuriously)
    * |uuid|: UUID (universally unique identifier)
    * |mattermost|: Mattermost specific names. Combination of team name, channel name and random text in public and private channels; personal meeting name in direct and group messages channels.
    * |ask|: The plugin asks you to select the name every time you start a meeting`,
		},
	})

	text := helpTitle + strings.ReplaceAll(commandHelp, "|", "`")
	post := &model.Post{
		UserId:    p.botID,
		ChannelId: args.ChannelId,
		Message:   text,
		RootId:    args.RootId,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)

	return &model.CommandResponse{}, nil
}

func (p *Plugin) settingsError(userID string, channelID string, errorText string, rootID string) (*model.CommandResponse, *model.AppError) {
	post := &model.Post{
		UserId:    p.botID,
		ChannelId: channelID,
		Message:   errorText,
		RootId:    rootID,
	}
	_ = p.API.SendEphemeralPost(userID, post)

	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeSettingsCommand(_ *plugin.Context, args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	l := p.b.GetUserLocalizer(args.UserId)
	text := ""

	userConfig, err := p.getUserConfig(args.UserId)
	if err != nil {
		mlog.Debug("Unable to get user config", mlog.Err(err))
		return p.settingsError(args.UserId, args.ChannelId, p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "jitsi.command.settings.unable_to_get",
				Other: "Unable to get user settings",
			},
		}), args.RootId)
	}

	if len(parameters) == 0 || parameters[0] == jitsiSettingsSeeCommand {
		text = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID: "jitsi.command.settings.current_values",
				Other: `###### Jitsi Settings:
* Embedded: |{{.Embedded}}|
* Show Pre-join Page: |{{.ShowPrejoinPage}}|
* Naming Scheme: |{{.NamingScheme}}|`,
			},
			TemplateData: map[string]string{
				"Embedded":        fmt.Sprintf("%v", userConfig.Embedded),
				"ShowPrejoinPage": fmt.Sprintf("%v", userConfig.ShowPrejoinPage),
				"NamingScheme":    userConfig.NamingScheme,
			},
		})
		post := &model.Post{
			UserId:    p.botID,
			ChannelId: args.ChannelId,
			Message:   strings.ReplaceAll(text, "|", "`"),
			RootId:    args.RootId,
		}
		_ = p.API.SendEphemeralPost(args.UserId, post)

		return &model.CommandResponse{}, nil
	}

	if len(parameters) != 2 {
		return p.settingsError(args.UserId, args.ChannelId, p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "jitsi.command.settings.invalid_parameters",
				Other: "Invalid settings parameters",
			},
		}), args.RootId)
	}

	switch parameters[0] {
	case commandArgEmbedded:
		switch parameters[1] {
		case valueTrue:
			userConfig.Embedded = true
		case valueFalse:
			userConfig.Embedded = false
		default:
			text = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "jitsi.command.settings.wrong_embedded_value",
					Other: "Invalid `embedded` value, use `true` or `false`.",
				},
			})
			userConfig = nil
		}
	case commandArgNamingScheme:
		switch parameters[1] {
		case jitsiNameSchemeAsk:
			userConfig.NamingScheme = "ask"
		case jitsiNameSchemeWords:
			userConfig.NamingScheme = "words"
		case jitsiNameSchemeUUID:
			userConfig.NamingScheme = "uuid"
		case jitsiNameSchemeMattermost:
			userConfig.NamingScheme = "mattermost"
		default:
			text = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "jitsi.command.settings.wrong_naming_scheme_value",
					Other: "Invalid `naming_scheme` value, use `ask`, `words`, `uuid` or `mattermost`.",
				},
			})
			userConfig = nil
		}
	case commandArgShowPrejoinPage:
		switch parameters[1] {
		case valueTrue:
			userConfig.ShowPrejoinPage = true
		case valueFalse:
			userConfig.ShowPrejoinPage = false
		default:
			text = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "jitsi.command.settings.wrong_show_prejoin_page_value",
					Other: "Invalid `show_prejoin_page` value, use `true` or `false`.",
				},
			})
			userConfig = nil
		}
	default:
		text = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "jitsi.command.settings.wrong_field",
				Other: "Invalid config field, use `embedded`, `show_prejoin_page` or `naming_scheme`.",
			},
		})
		userConfig = nil
	}

	if userConfig == nil {
		return p.settingsError(args.UserId, args.ChannelId, text, args.RootId)
	}

	err = p.setUserConfig(args.UserId, userConfig)
	if err != nil {
		mlog.Debug("Unable to set user settings", mlog.Err(err))
		return p.settingsError(args.UserId, args.ChannelId, p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "jitsi.command.settings.unable_to_set",
				Other: "Unable to set user settings",
			},
		}), args.RootId)
	}

	post := &model.Post{
		UserId:    p.botID,
		ChannelId: args.ChannelId,
		Message: p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "jitsi.command.settings.updated",
				Other: fmt.Sprintf("Jitsi settings updated:\n\n* %s: `%s`", parameters[0], parameters[1]),
			},
		}),
		RootId: args.RootId,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)

	return &model.CommandResponse{}, nil
}
