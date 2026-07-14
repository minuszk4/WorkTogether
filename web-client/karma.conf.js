module.exports = (config) => config.set({
  basePath: '',
  frameworks: ['jasmine', '@angular-devkit/build-angular'],
  plugins: [
    require('karma-jasmine'),
    require('karma-chrome-launcher'),
    require('karma-jasmine-html-reporter'),
    require('karma-coverage'),
    require('@angular-devkit/build-angular/plugins/karma'),
  ],
  customLaunchers: {
    ChromeHeadlessCI: { base: 'ChromeHeadless', flags: ['--no-sandbox'] },
  },
  reporters: ['progress', 'kjhtml'],
  browsers: ['ChromeHeadlessCI'],
  restartOnFileChange: true,
});
