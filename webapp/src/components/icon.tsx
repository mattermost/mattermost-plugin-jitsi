import React from 'react';
import Svgs from 'constants/svgs';

export default class Icon extends React.PureComponent {
    render() {
        const style = getStyle();
        return (
            <span
                style={style.iconStyle}
                className='icon'
                aria-hidden='true'
                dangerouslySetInnerHTML={{__html: Svgs.VIDEO_CAMERA}}
            />
        );
    }
}

function getStyle(): { [key: string]: React.CSSProperties } {
    return {
        iconStyle: {
            position: 'relative',
            top: '-1px'
        }
    };
}
