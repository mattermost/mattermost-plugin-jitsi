// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import ReactDOM from 'react-dom';
import {Provider} from 'react-redux';

import Conference from './conference/';
import I18nProvider from './i18n_provider/';

export class InjectionProvider extends React.Component<any> {
    public render(): React.ReactElement {
        const stores = {...this.props};
        delete stores.children;
        return React.createElement(Provider as any, stores, this.props.children);
    }
}

export default class RootPortal {
    el: HTMLElement;
    store: any;

    constructor(registry: any, store: any) {
        this.el = document.createElement('div');
        this.store = store;
        const rootPortal = document.getElementById('root-portal');
        if (rootPortal) {
            rootPortal.appendChild(this.el);
        } else {
            registry.registerRootComponent(Conference);
        }
    }

    cleanup() {
        const rootPortal = document.getElementById('root-portal');
        if (rootPortal) {
            rootPortal.removeChild(this.el);
        }
    }

    render() {
        const rootPortal = document.getElementById('root-portal');
        if (rootPortal) {
            ReactDOM.render((
                <InjectionProvider store={this.store}>
                    <I18nProvider>
                        <Conference/>
                    </I18nProvider>
                </InjectionProvider>
            ), this.el);
        }
    }
}
