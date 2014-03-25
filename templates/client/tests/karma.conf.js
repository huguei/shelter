module.exports = function(config) {
  config.set({
    basePath: '..',
    frameworks: ['jasmine'],

    files: [
      'js/angular.min.js',
      'tests/angular-mocks.js',
      'js/angular-animate.min.js',
      'js/angular-cookies.min.js',
      'js/angular-translate.min.js',
      'js/angular-translate-loader-static-files.min.js',
      'js/angular-translate-storage-cookie.min.js',
      'js/angular-translate-storage-local.min.js',
      'js/moment-with-langs.min.js',
      'js/shelter.js',
      'tests/statics.js',
      'tests/domain.js',
      'directives/*.html'
    ],

    exclude: [],

    preprocessors: {
      'directives/*.html': ['ng-html2js']
    },

    ngHtml2JsPreprocessor: {
      stripPrefix: '.*/shelter/templates/client',
      prependPrefix: '/',
      moduleName: 'directives'
    },

    reporters: ['progress'],
    port: 9876,
    colors: true,
    logLevel: config.LOG_INFO,
    autoWatch: true,
    browsers: ['Firefox'],
    singleRun: false
  });
};