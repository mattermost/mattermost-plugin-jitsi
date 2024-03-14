var path = require('path');

const HtmlWebPackPlugin = require('html-webpack-plugin');

module.exports = [
    Object.assign({}, {
        entry: [
            './src/index.tsx'
        ],
        resolve: {
            modules: [
                'src',
                'node_modules'
            ],
            extensions: ['*', '.js', '.jsx', '.ts', '.tsx']
        },
        module: {
            rules: [
                {
                    test: /\.(js|jsx|ts|tsx)$/,
                    exclude: /node_modules/,
                    use: {
                        loader: 'babel-loader',
                        options: {
                            cacheDirectory: true

                            // Babel configuration is in babel.config.js because jest requires it to be there.
                        }
                    }
                }
            ]
        },
        externals: {
            react: 'React',
            'react-dom': 'ReactDOM',
            'react-intl': 'ReactIntl',
            redux: 'Redux',
            'react-redux': 'ReactRedux',
            'prop-types': 'PropTypes',
            'react-bootstrap': 'ReactBootstrap',
            'react-router-dom': 'ReactRouterDom'
        },
        output: {
            path: path.join(__dirname, '/dist'),
            publicPath: '/',
            filename: 'main.js'
        }
    }),
    Object.assign({}, {
        entry: [
            './src/jaas/index.tsx'
        ],
        resolve: {
            modules: [
                'src/jaas',
                'node_modules'
            ],
            extensions: ['*', '.js', '.jsx', '.ts', '.tsx']
        },
        output: {
            path: path.join(__dirname, '../public/jaas'),
            publicPath: '/plugins/jitsi/public/jaas',
            filename: 'jaas-main.js'
        },
        module: {
            rules: [
                {
                    test: /\.(js|jsx|ts|tsx)$/,
                    exclude: /node_modules/,
                    use: {
                        loader: 'babel-loader',
                        options: {
                            cacheDirectory: true

                            // Babel configuration is in babel.config.js because jest requires it to be there.
                        }
                    }
                }
            ]
        },
        plugins: [
            new HtmlWebPackPlugin({
                template: './src/jaas/index.html',
                filename: 'jaas.html',
                inject: true
            })
        ]
    })
];
