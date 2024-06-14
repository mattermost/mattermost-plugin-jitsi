import React, {useEffect} from 'react';
import {useDispatch} from 'react-redux';
import {loadConfig} from './actions';

const App = (): React.ReactNode => {
    const dispatch = useDispatch();

    useEffect(() => {
        dispatch(loadConfig());
    }, []);

    // This container is used just for making the API call for fetching the config, it doesn't render anything.
    return null;
};

export default App;
