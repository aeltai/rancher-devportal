const config = require('@rancher/shell/vue.config'); // eslint-disable-line @typescript-eslint/no-var-requires

const vueConfig = config(__dirname, {
  excludes: [],
  proxy: {
    '/devportal-api': {
      target:       'http://localhost:9010',
      changeOrigin: true,
      pathRewrite:  { '^/devportal-api': '' },
    },
  },
});

vueConfig.lintOnSave = false;

vueConfig.devServer = {
  ...(vueConfig.devServer || {}),
  historyApiFallback: true,
};

const existingChainWebpack = vueConfig.chainWebpack;

vueConfig.chainWebpack = (webpackConfig) => {
  if (existingChainWebpack) {
    existingChainWebpack(webpackConfig);
  }

  webpackConfig.plugins.delete('eslint');
};

module.exports = vueConfig;
