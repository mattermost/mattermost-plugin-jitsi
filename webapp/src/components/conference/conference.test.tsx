import React from 'react';
import {jest, describe, expect, it} from '@jest/globals';
import {shallow} from 'enzyme';

import {Post} from 'mattermost-redux/types/posts';

import Conference from './conference';

describe('Conference', () => {
    const basePost: Post = {
        id: 'test',
        create_at: 100,
        update_at: 100,
        edit_at: 100,
        delete_at: 100,
        message: 'test-message',
        is_pinned: false,
        user_id: 'test-user-id',
        channel_id: 'test-channel-id',
        root_id: '',
        parent_id: '',
        original_id: '',
        type: 'custom_jitsi',
        hashtags: '',
        props: {
            jwt_meeting_valid_until: 123,
            meeting_link: 'http://test-meeting-link/test',
            jwt_meeting: true,
            meeting_jwt: 'xxxxxxxxxxxx',
            meeting_topic: 'Test topic',
            meeting_id: 'test_meeting_id',
            meeting_personal: false
        }
    };

    const actions = {
        openJitsiMeeting: jest.fn(),
        setUserStatus: jest.fn()
    };

    const defaultProps = {
        post: basePost,
        jwt: null,
        actions,
        currentUserId: 'test'
    };

    Conference.prototype.getViewportWidth = () => 10;
    Conference.prototype.getViewportHeight = () => 10;
    Conference.prototype.componentDidUpdate = jest.fn();
    Conference.prototype.componentDidMount = jest.fn();
    Conference.prototype.componentWillUnmount = jest.fn();

    it('should render null if the post type is null', () => {
        const props = {...defaultProps, post: null};
        const wrapper = shallow(
            <Conference {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    it('should render and initialize the conference interface', () => {
        const wrapper = shallow(
            <Conference {...defaultProps}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    describe('should show the correct buttons depending on the state', () => {
        const wrapper = shallow(
            <Conference {...defaultProps}/>
        );
        const instance = wrapper.instance();

        it('should have down, open outside, maximize and close buttons', () => {
            instance.setState({minimized: true, position: 'top'});
            expect(wrapper).toMatchSnapshot();
        });

        it('should have up, open outside, maximize and close buttons', () => {
            instance.setState({minimized: true, position: 'bottom'});
            expect(wrapper).toMatchSnapshot();
        });

        it('should have open outside, minimize and close buttons', () => {
            instance.setState({minimized: false});
            expect(wrapper).toMatchSnapshot();
        });
    });

    describe('should show the the loading spinner depending on the state', () => {
        const wrapper = shallow(
            <Conference {...defaultProps}/>
        );
        const instance = wrapper.instance();

        it('should show loading', () => {
            instance.setState({loading: true});
            expect(wrapper).toMatchSnapshot();
        });

        it('should not show loading', () => {
            instance.setState({loading: false});
            expect(wrapper).toMatchSnapshot();
        });
    });

    describe('should maximize based on the state', () => {
        const wrapper = shallow(
            <Conference {...defaultProps}/>
        );
        const instance = wrapper.instance() as Conference;

        it('should toggle tile only if was open before minimize and now is closed', () => {
            instance.api = {
                executeCommand: jest.fn()
            };
            instance.setState({wasTileView: true, isTileView: true});
            instance.maximize();
            expect(instance.api.executeCommand).not.toBeCalledWith('toggleTileView');
            instance.api.executeCommand.mockClear();

            instance.setState({wasTileView: true, isTileView: false});
            instance.maximize();
            expect(instance.api.executeCommand).toBeCalledWith('toggleTileView');
            instance.api.executeCommand.mockClear();

            instance.setState({wasTileView: false, isTileView: true});
            instance.maximize();
            expect(instance.api.executeCommand).toBeCalledWith('toggleTileView');
            instance.api.executeCommand.mockClear();

            instance.setState({wasTileView: false, isTileView: false});
            instance.maximize();
            expect(instance.api.executeCommand).not.toBeCalledWith('toggleTileView');
            instance.api.executeCommand.mockClear();
        });

        it('should toggle filmstrip only if was open before minimize and now is closed', () => {
            instance.api = {
                executeCommand: jest.fn()
            };
            instance.setState({wasFilmStrip: true, isFilmStrip: true});
            instance.maximize();
            expect(instance.api.executeCommand).not.toBeCalledWith('toggleFilmStrip');
            instance.api.executeCommand.mockClear();

            instance.setState({wasFilmStrip: true, isFilmStrip: false});
            instance.maximize();
            expect(instance.api.executeCommand).toBeCalledWith('toggleFilmStrip');
            instance.api.executeCommand.mockClear();

            instance.setState({wasFilmStrip: false, isFilmStrip: true});
            instance.maximize();
            expect(instance.api.executeCommand).toBeCalledWith('toggleFilmStrip');
            instance.api.executeCommand.mockClear();

            instance.setState({wasFilmStrip: false, isFilmStrip: false});
            instance.maximize();
            expect(instance.api.executeCommand).not.toBeCalledWith('toggleFilmStrip');
            instance.api.executeCommand.mockClear();
        });
    });

    describe('should minimize based on the state', () => {
        const wrapper = shallow(
            <Conference {...defaultProps}/>
        );
        const instance = wrapper.instance() as Conference;

        it('should toggle tile only if is open before minimizing', () => {
            instance.api = {
                executeCommand: jest.fn()
            };
            instance.setState({isTileView: true});
            instance.minimize();
            expect(instance.api.executeCommand).toBeCalledWith('toggleTileView');
            instance.api.executeCommand.mockClear();

            instance.setState({isTileView: false});
            instance.minimize();
            expect(instance.api.executeCommand).not.toBeCalledWith('toggleTileView');
            instance.api.executeCommand.mockClear();
        });

        it('should toggle filmstrip only if is open before minimize', () => {
            instance.api = {
                executeCommand: jest.fn()
            };
            instance.setState({isFilmStrip: true});
            instance.minimize();
            expect(instance.api.executeCommand).toBeCalledWith('toggleFilmStrip');
            instance.api.executeCommand.mockClear();

            instance.setState({isFilmStrip: false});
            instance.minimize();
            expect(instance.api.executeCommand).not.toBeCalledWith('toggleFilmStrip');
            instance.api.executeCommand.mockClear();
        });
    });

    it('should execute the hangup command, wait and call the action to close the meeting, and reset the state on closed', (done) => {
        defaultProps.actions.openJitsiMeeting.mockClear();
        const wrapper = shallow(
            <Conference {...defaultProps}/>
        );
        const instance = wrapper.instance() as Conference;
        instance.setState({
            minimized: false,
            loading: false,
            position: 'top',
            wasTileView: false,
            isTileView: false,
            wasFilmStrip: false,
            isFilmStrip: false
        });
        instance.api = {
            executeCommand: jest.fn(),
            dispose: jest.fn()
        };
        instance.close();
        expect(defaultProps.actions.openJitsiMeeting).not.toBeCalled();
        expect(instance.api.executeCommand).toBeCalledWith('hangup');
        setTimeout(() => {
            expect(defaultProps.actions.openJitsiMeeting).toBeCalledTimes(1);
            expect(defaultProps.actions.openJitsiMeeting).toBeCalledWith(null, null);
            expect(instance.api.dispose).toBeCalled();
            expect(instance.state).toEqual({
                minimized: true,
                loading: true,
                position: 'bottom',
                wasTileView: true,
                isTileView: true,
                wasFilmStrip: true,
                isFilmStrip: true
            });

            if (done) {
                done();
            }
        }, 210);
    });
});
