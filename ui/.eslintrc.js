module.exports = {
  extends: ['eslint:recommended', 'next/core-web-vitals', 'prettier'],
  rules: {
    '@next/next/no-img-element': 'off',
    'react-hooks/exhaustive-deps': 'off',
  },
  plugins: ['jest', '@typescript-eslint'],
  overrides: [
    {
      files: ['*.ts', '*.tsx'],
      extends: [
        'plugin:@typescript-eslint/recommended',
        'plugin:@typescript-eslint/recommended-requiring-type-checking',
      ],
      parserOptions: {
        tsconfigRootDir: __dirname,
        project: ['./tsconfig.json'],
      },
    },
  ],
  ignorePatterns: ['/*.*'],
  globals: {
    Promise: true,
    Set: true,
    jsonBody: true,
  },
  env: {
    'jest/globals': true,
    es6: true,
  },
}
