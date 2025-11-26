const path = require('path');
const webpack = require('webpack');
const TerserPlugin = require('terser-webpack-plugin');

module.exports = env => {
    return {
        target: 'node',
        entry: {
            cli: path.join(__dirname, 'src', 'cli', 'index.ts'),
        },
        output: {
            path: path.join(__dirname, 'dist'),
            filename: '[name].js',
        },
        resolve: {
            extensions: ['.tsx', '.ts', '.js'],
        },
        module: {
            rules: [
                {
                    test: /\.ts$/,
                    include: /src/,
                    use: [{ loader: 'ts-loader' }],
                },
            ],
        },
        optimization: {
            minimize: true,
            minimizer: [
                new TerserPlugin({
                    terserOptions: {
                        mangle: false,
                    },
                }),
            ],
        },
        plugins: [
            new webpack.DefinePlugin({
                'process.env.TARGET_ENV': JSON.stringify(
                    env.targetEnv || 'PROD',
                ),
            }),
            new webpack.BannerPlugin({
                banner: '#!/usr/bin/env node',
                raw: true,
            }),
        ],
        externals: {
            // Don't bundle native modules
            'fsevents': 'commonjs fsevents',
        },
    };
};
