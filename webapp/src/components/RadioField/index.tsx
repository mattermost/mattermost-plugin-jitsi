import React, {ReactNode} from 'react';

import {RadioOptionsType} from 'types';

type Props = {
    heading: string | ReactNode;
    isInline: boolean;
    options: RadioOptionsType[];
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    description: string | ReactNode;
}

export const RadioField = ({heading, isInline, options, onChange, description}: Props) => {
    return (
        <div className='form-group'>
            <label className='col-sm-4'>
                {heading}
            </label>
            <div className='col-sm-8'>
                <div className={isInline ? '' : 'radio'}>
                    {options.map((option) => (
                        <label
                            className={isInline ? 'pt-0 radio-inline' : 'pt-0'}
                            key={option.value}
                        >
                            <input
                                type='radio'
                                value={option.value}
                                checked={option.checked}
                                onChange={onChange}
                            />
                            {option.label}
                        </label>
                    ))}
                </div>
                <div className='help-text'>
                    <span>
                        {description}
                    </span>
                </div>
            </div>
        </div>
    );
};
