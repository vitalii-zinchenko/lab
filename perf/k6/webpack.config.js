const path = require('path');

module.exports = {
  mode: 'production',
  entry: {
    smoke:  './src/smoke.ts',
    load:   './src/load.ts',
    stress:        './src/stress.ts',
    'insert-usage': './src/insert-usage.ts',
  },
  output: {
    path: path.resolve(__dirname, 'dist'),
    libraryTarget: 'commonjs',
    filename: '[name].js',
  },
  resolve: {
    extensions: ['.ts', '.js'],
  },
  module: {
    rules: [{
      test: /\.ts$/,
      use: { loader: 'ts-loader', options: { transpileOnly: true } },
      exclude: /node_modules/,
    }],
  },
  // k6 runs in a browser-like JS environment
  target: 'web',
  // Prevent webpack from bundling k6 built-in modules and remote imports
  externals: /^(k6|https?:\/\/)(\/.*)?/,
  stats: { colors: true },
  // Keep output readable — k6 errors show line numbers
  optimization: { minimize: false },
};
