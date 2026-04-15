import { globalIgnores } from 'eslint/config'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import pluginOxlint from 'eslint-plugin-oxlint'

import stylistic from '@stylistic/eslint-plugin'

const INLINE_ELEMENTS = [
  'a',
  'abbr',
  'audio',
  'b',
  'bdi',
  'bdo',
  'canvas',
  'cite',
  'code',
  'data',
  'del',
  'dfn',
  'em',
  'i',
  'iframe',
  'ins',
  'kbd',
  'label',
  'map',
  'mark',
  'noscript',
  'object',
  'output',
  'picture',
  'q',
  'ruby',
  's',
  'samp',
  'small',
  'span',
  'strong',
  'sub',
  'sup',
  'svg',
  'time',
  'u',
  'var',
  'video',
  'button',
]

// To allow more languages other than  in  files, uncomment the following lines:
// import { configureVueProject } from '@vue/eslint-config-typescript'
// configureVueProject({ scriptLangs: ['ts', 'tsx'] })
// More info at https://github.com/vuejs/eslint-config-typescript/#advanced-setup

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },

  globalIgnores(['**/dist/**', '**/dist-ssr/**', '**/coverage/**', '**/extension/**']),

  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  ...pluginOxlint.configs['flat/recommended'],


  {
    linterOptions: {
      reportUnusedDisableDirectives: 'off',
    },
    ignores: ['src/wailsjs'],
    plugins: {
      '@stylistic': stylistic,
    },
    rules: {
      // stylistic 规则列表
      // https://eslint.style/rules
      '@stylistic/space-infix-ops': ['error'],
      '@stylistic/eol-last': ['error', 'always'],
      '@stylistic/spaced-comment': ['error', 'always'],
      '@stylistic/lines-around-comment': ['error', {
        beforeBlockComment: true,
      }],
      '@stylistic/no-multiple-empty-lines': ['error', { max: 1, maxEOF: 0 }],
      '@stylistic/object-curly-spacing': ['error', 'always'],
      '@stylistic/no-trailing-spaces': ['error', { ignoreComments: true }],
      '@stylistic/comma-spacing': ['error', { before: false, after: true }],
      '@stylistic/comma-style': ['error', 'last'],
      '@stylistic/quote-props': ['error', 'as-needed'],
      '@stylistic/indent': ['error', 2],
      '@stylistic/space-in-parens': ['error', 'never'],
      '@stylistic/no-extra-semi': 'error',
      '@stylistic/no-multi-spaces': 'error',
      '@stylistic/array-bracket-spacing': ['error', 'never'],
      semi: ['error', 'never'],
      quotes: ['error', 'single'],
      indent: ['error', 2, { SwitchCase: 1 }],
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': ['error'],
      'no-shadow': 'off',
      'arrow-parens': ['error', 'as-needed'],
      'no-confusing-arrow': ['error', { allowParens: true, onlyOneSimpleParam: true }],
      // eslint-plugin-vue 规则列表
      // https://eslint.vuejs.org/rules/
      '@stylistic/comma-dangle': [
        'error',
        {
          arrays: 'always-multiline',
          objects: 'always-multiline',
          imports: 'always-multiline',
          exports: 'always-multiline',
          functions: 'always-multiline',
        },
      ],
      'vue/no-unused-components': 0,
      'vue/html-closing-bracket-spacing': 'error',
      'vue/html-indent': ['error', 2],
      'vue/script-indent': ['error', 2, { switchCase: 1 }],
      'vue/mustache-interpolation-spacing': [2, 'always'],
      'vue/max-len': [
        'error',
        {
          code: 200,
          template: 200,
          tabWidth: 2,
          comments: 400,
          // ignorePattern: INLINE_ELEMENTS,
          ignoreHTMLAttributeValues: false,
          ignoreHTMLTextContents: true,
          ignoreComments: true,
          ignoreStrings: true,
          ignoreTrailingComments: true,
          ignoreUrls: true,
          ignoreTemplateLiterals: true,
          ignoreRegExpLiterals: true,
        },
      ],
      'vue/singleline-html-element-content-newline': [
        'error',
        {
          ignoreWhenNoAttributes: false,
          ignoreWhenEmpty: true,
          ignores: [...INLINE_ELEMENTS],
        },
      ],
      'vue/max-attributes-per-line': [
        'error',
        {
          singleline: 3,
        // multiline: {
        //     max: 3,
        //     allowFirstLine: true,
        // },
        },
      ],
      'vue/first-attribute-linebreak': ['error', {
        singleline: 'beside',
        multiline: 'below',
      }],
      'vue/html-closing-bracket-newline': ['error', {
        singleline: 'never',
        multiline: 'always',
        selfClosingTag: {
          singleline: 'never',
          multiline: 'always',
        },
      },
      ],
      'vue/multiline-html-element-content-newline': ['error', {
        ignoreWhenEmpty: true,
        ignores: ['pre', 'textarea', ...INLINE_ELEMENTS],
        allowEmptyLines: false,
      }],
      'vue/no-v-html': 0,
      'vue/require-prop-types': 0,
      'vue/prop-name-casing': 0,
      'vue/multi-word-component-names': 0,
      'vue/block-tag-newline': ['error', {
        singleline: 'always',
        multiline: 'always',
        maxEmptyLines: 0,
      }],
      // 'import/no-extraneous-dependencies': [
      //   'error',
      //   {
      //     devDependencies: true,
      //     optionalDependencies: true,
      //     peerDependencies: true,
      //     bundledDependencies: true,
      //   },
      // ],
    },
  },
)
