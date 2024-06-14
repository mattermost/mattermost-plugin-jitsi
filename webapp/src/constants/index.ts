const MEETING_ID = 'meetingID';
const MISMATCH_SERVER_TYPE_MESSAGE = 'You cannot open a Jaas meeting when the selected server type is Jitsi.';
const JAAS_EPHEMERAL_MESSAGE = `${MISMATCH_SERVER_TYPE_MESSAGE} Please contact your system administrator.`;
const JAAS_ADMIN_EPHEMERAL_MESSAGE = `${MISMATCH_SERVER_TYPE_MESSAGE} You can update the server type by going to "System Console > Plugins > Jitsi".`;

export default {
    MEETING_ID,
    MISMATCH_SERVER_TYPE_MESSAGE,
    JAAS_EPHEMERAL_MESSAGE,
    JAAS_ADMIN_EPHEMERAL_MESSAGE
};
