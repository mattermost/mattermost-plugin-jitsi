import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';

import {GenericAction} from 'mattermost-redux/types/actions';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/common';

import {GlobalState, plugin} from 'types';
import {openJitsiMeeting, setUserStatus} from 'actions';
import manifest from 'manifest';
import Conference from './conference';

function mapStateToProps(state: GlobalState) {
    const config = state[`plugins-${manifest.id}` as plugin].config;

    return {
        currentUser: getCurrentUser(state),
        post: state[`plugins-${manifest.id}` as plugin].openMeeting,
        jwt: state[`plugins-${manifest.id}` as plugin].openMeetingJwt,
        showPrejoinPage: config.show_prejoin_page,
        meetingEmbedded: config.embedded
    };
}

function mapDispatchToProps(dispatch: Dispatch<GenericAction>) {
    return {
        actions: bindActionCreators({
            openJitsiMeeting,
            setUserStatus
        }, dispatch)
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(Conference);
