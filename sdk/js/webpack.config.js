const webpack = require('webpack');
require('dotenv').config({path: '../../.env'});

module.exports = {
  entry: './src/index.js',
  devtool: 'source-map',//eval | source-map
  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        exclude: /node_modules/,
        use: ['babel-loader']
      },
      {
        test: /\.(scss|less|css)$/,
        use: ["style-loader", "css-loader", "sass-loader"]
      },
    ]
  },
  resolve: {
    extensions: ['*', '.js', '.jsx']
  },
  output: {
    path: __dirname + '/dist',
    publicPath: '/',
    filename: 'ion-sdk.js'
  },
  plugins: [
    new webpack.HotModuleReplacementPlugin(),
    new webpack.DefinePlugin({
      'process.env.WS_PORT': process.env.WS_PORT
    })
  ],
  devServer: {
    contentBase: './dist',
    hot: true,
    host: "0.0.0.0",
    port: process.env.HTTP_PORT,
    onListening: function(server) {
      const port = server.listeningApp.address().port;
      console.log('Listening on port:', port);
    }
  }
};
