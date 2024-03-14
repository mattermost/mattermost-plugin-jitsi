import React from 'react';
import {Client4} from 'mattermost-redux/client';
import {id as pluginId} from '../../../manifest';
import constants from '../../../constants';

type Props = {}

type State = {
    jaasJwt?: string | null,
    jaasRoom?: string | null,
}

const JWT = 'jwt';
const JAAS_URL = '8x8.vc';

export default class JaaSConference extends React.PureComponent<Props, State> {
    api: any;
    style = getStyle();

    constructor(props: Props) {
        super(props);
        this.state = {};
    }

    initJaaS(jwt: string, room: string) {
        const ws = Math.max(document.documentElement.clientWidth ?? 0, window?.innerWidth ?? 0);
        const hs = Math.max(document.documentElement.clientHeight ?? 0, window?.innerHeight ?? 0);
        const options = {
            width: ws,
            height: hs,
            roomName: room,
            jwt,
            parentNode: document.querySelector('#jitsiMeet')
        };

        this.api = new (window as any).JitsiMeetExternalAPI(JAAS_URL, options);
    }

    startJaaSMeetingWindow(jwt: string | null, meetingId: string | null) {
        const options = {
            method: 'post',
            body: JSON.stringify({jaasJwt: jwt ?? '', jaasPath: meetingId}),
            headers: {}
        };

        const baseUrl = `/plugins/${pluginId}`;
        fetch(`${baseUrl}/api/v1/meetings/jaas/settings`, Client4.getOptions(options)).
            then((result) => {
                if (result.ok) {
                    return result.json();
                }

                throw new Error(result.statusText);
            }).
            then((result) => {
                this.setState({
                    jaasRoom: result.jaasRoom,
                    jaasJwt: result.jaasJwt
                });
            });
    }

    componentDidMount() {
        if (!(window as any).JitsiMeetExternalAPI) {
            // Check if the script is already loaded or not
            if (document.getElementById(`${pluginId}_jitsi_meet_external_api_script`)) {
                return;
            }
            const script = document.createElement('script');
            script.type = 'text/javascript';
            script.id = `${pluginId}_jitsi_meet_external_api_script`;
            script.onload = () => {
                const params = new URLSearchParams(window.location.search);
                const jwt = params.get(JWT);
                const meetingId = params.get(constants.MEETING_ID);

                this.startJaaSMeetingWindow(jwt, meetingId);
            };
            script.src = `${(window as any).location.origin}/plugins/${pluginId}/jitsi_meet_external_api.js`;
            document.head.appendChild(script);
        }
    }

    componentDidUpdate(prevProps: Props, prevState: State) {
        if (prevState.jaasJwt !== this.state.jaasJwt || prevState.jaasRoom !== this.state.jaasRoom) {
            this.initJaaS(this.state.jaasJwt as string, this.state.jaasRoom as string);
        }
    }

    render() {
        return (
            <div
                id='jitsiMeet'
                style={this.style.jitsiMeetContainer}
            />
        );
    }
}

function getStyle() : { [key: string]: React.CSSProperties} {
    return {
        jitsiMeetContainer: {
            width: '100%',
            height: '100%',
            display: 'flex'
        }
    };
}
