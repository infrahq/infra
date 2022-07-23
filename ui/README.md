# Infra Dashboard

This directory contains the source code for the Infra Dashboard.

## Set up

- Install the latest version of Node.js `brew install node`
- Install the following extensions if using Visual Studio Code:
  - [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint)
  - [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode)
  - [Tailwind CSS IntelliSense](https://marketplace.visualstudio.com/items?itemName=bradlc.vscode-tailwindcss)
  - Enable **Format on Save** in settings

## Develop

```
npm install
npm run dev
```

## Build

```
npm run build
npm run export -- -o ../internal/server/ui/static
```

Now building the Infra Server will include the Infra Dashboard.

## Linting

Linting is done via [ESLint](https://eslint.org/)

```
npm run lint
```

## Formatting

Code is formatted using [Prettier](https://prettier.io/)

To check for issues:

```
npm run format
```

To fix:

```
npm run format:fix
```
