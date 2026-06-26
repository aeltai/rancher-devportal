const config = require('@rancher/shell/vue.config'); // eslint-disable-line @typescript-eslint/no-var-requires

const vueConfig = config(__dirname, {
  excludes: [],
});

vueConfig.lintOnSave = false;

const existingChainWebpack = vueConfig.chainWebpack;

vueConfig.chainWebpack = (webpackConfig) => {
  if (existingChainWebpack) {
    existingChainWebpack(webpackConfig);
  }

  webpackConfig.plugins.delete('eslint');
};

module.exports = vueConfig;
