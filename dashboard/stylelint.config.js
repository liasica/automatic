/** @type {import('stylelint').Config} */
export default {
  plugins: [
    'stylelint-order',
    '@stylistic/stylelint-plugin',
  ],
  overrides: [
    {
      files: ['**/*.scss'],
      customSyntax: 'postcss-scss',
    },
  ],
  extends: [
    'stylelint-config-standard',
    'stylelint-config-standard-scss',
    'stylelint-config-recommended',
    'stylelint-config-recommended-scss',
    'stylelint-config-recommended-vue/scss',
  ],
  rules: {
    'block-no-empty': true,
    'property-no-unknown': [
      true,
      {
        ignoreProperties: [
          '/d/',
        ],
      },
    ],
    'at-rule-no-unknown': null,
    'selector-pseudo-element-no-unknown': [
      true,
      {
        ignorePseudoElements: [
          'deep',
        ],
      },
    ],
    'selector-pseudo-class-no-unknown': [
      true,
      {
        ignorePseudoClasses: [
          'global',
          'deep',
        ],
      },
    ],
    'scss/dollar-variable-pattern': [
      /^[\s\S]/, { ignore: 'global' },
    ],
    'scss/no-global-function-names': null,

    // stylistic rules from @stylistic/stylelint-plugin
    // https://github.com/stylelint-stylistic/stylelint-stylistic/blob/main/docs/user-guide/rules.md
    '@stylistic/indentation': 2,
    '@stylistic/max-empty-lines': 1,
    '@stylistic/no-empty-first-line': true,
    '@stylistic/no-eol-whitespace': true,
    '@stylistic/no-missing-end-of-source-newline': true,
    '@stylistic/declaration-block-semicolon-space-before': 'never',
  },
}
