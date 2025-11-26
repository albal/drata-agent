const path = require('path');
const webpack = require('webpack');
const TerserPlugin = require('terser-webpack-plugin');
const packageJson = require('./package.json');

module.exports = env => {
    // Allow external node_modules path for building without full deps
    const nodeModulesPath = process.env.CLI_NODE_MODULES || path.join(__dirname, 'node_modules');

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
            modules: [nodeModulesPath, 'node_modules'],
        },
        resolveLoader: {
            modules: [nodeModulesPath, 'node_modules'],
        },
        module: {
            rules: [
                {
                    test: /\.ts$/,
                    include: [
                        path.join(__dirname, 'src', 'cli'),
                        path.join(__dirname, 'src', 'enums'),
                    ],
                    use: [{
                        loader: 'ts-loader',
                        options: {
                            configFile: 'tsconfig.cli.json',
                        },
                    }],
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
                '__CLI_VERSION__': JSON.stringify(packageJson.version),
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
