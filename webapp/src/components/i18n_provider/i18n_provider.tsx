import React from 'react';
import {IntlProvider} from 'react-intl';

import {getTranslations} from '../../../i18n';

export type Props = {
    currentLocale: string,
    children: React.ReactNode
}

export class I18nProvider extends React.PureComponent<Props> {
    render() {
        return (
            <IntlProvider
                locale={this.props.currentLocale}
                key={this.props.currentLocale}
                messages={getTranslations(this.props.currentLocale)}
            >
                {this.props.children}
            </IntlProvider>
        );
    }
}
