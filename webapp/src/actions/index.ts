import {PostTypes} from 'mattermost-redux/action_types';
import {DispatchFunc, GetStateFunc, ActionFunc, ActionResult} from 'mattermost-redux/types/actions';
import {Post} from 'mattermost-redux/types/posts';
import ActionTypes from 'action_types';

import Client from 'client';

export function startMeeting(channelId: string, personal: boolean = false, topic: string = '', meetingId: string = ''): ActionFunc {
    return async (dispatch: DispatchFunc, getState: GetStateFunc): Promise<ActionResult> => {
        try {
            await Client.startMeeting(channelId, personal, topic, meetingId);
        } catch (error) {
            const post: Post = {
                id: 'jitsiPlugin' + Date.now(),
                create_at: Date.now(),
                update_at: 0,
                edit_at: 0,
                delete_at: 0,
                is_pinned: false,
                user_id: getState().entities.users.currentUserId,
                channel_id: channelId,
                root_id: '',
                parent_id: '',
                original_id: '',
                reply_count: 0,
                message: 'We could not start a meeting at this time.',
                type: 'system_ephemeral',
                props: {},
                metadata: {
                    embeds: [],
                    emojis: [],
                    files: [],
                    images: {},
                    reactions: []
                },
                hashtags: '',
                pending_post_id: ''
            };

            dispatch({
                type: PostTypes.RECEIVED_POSTS,
                data: {
                    order: [],
                    posts: {
                        [post.id]: post
                    }
                },
                channelId
            });

            return {error};
        }

        return {data: true};
    };
}

export function enrichMeetingJwt(meetingJwt: string): ActionFunc {
    return async (): Promise<ActionResult> => {
        try {
            const data = await Client.enrichMeetingJwt(meetingJwt);
            return {data};
        } catch (error) {
            return {error};
        }
    };
}

export function loadConfig(): ActionFunc {
    return async (dispatch: DispatchFunc): Promise<ActionResult> => {
        try {
            const data = await Client.loadConfig();
            dispatch({
                type: ActionTypes.CONFIG_RECEIVED,
                data
            });
            return {data};
        } catch (error) {
            return {error};
        }
    };
}

export function openJitsiMeeting(post: Post | null, jwt: string | null): ActionFunc {
    return (dispatch: DispatchFunc): ActionResult => {
        dispatch({
            type: ActionTypes.OPEN_MEETING,
            data: {
                post,
                jwt
            }
        });
        return {data: null};
    };
}

export function setUserStatus(userId: string, status: string): ActionFunc {
    return async (dispatch: DispatchFunc): Promise<ActionResult> => {
        try {
            const data = await Client.setUserStatus(userId, status);
            dispatch({
                type: ActionTypes.USER_STATUS_CHANGED,
                data
            });
            return {data};
        } catch (error) {
            return {error};
        }
    };
}

export function sendEphemeralPost(message: string, channelID: string, userID: string): ActionFunc {
    return (dispatch: DispatchFunc): ActionResult => {
        const timestamp = Date.now();
        const post = {
            id: 'jitsi' + timestamp,
            user_id: userID,
            channel_id: channelID,
            message,
            type: 'system_ephemeral',
            create_at: timestamp,
            update_at: timestamp
        };

        dispatch({
            type: PostTypes.RECEIVED_NEW_POST,
            data: post,
            channelID
        });

        return {data: post};
    };
}
