module.exports = {
    extends: ['@commitlint/config-conventional'],
    rules: {
        'type-enum': [
            2,
            'always',
            [
                'feat',
                'fix',
                'improve',
                'maintain',
            ]
        ],
        'body-max-line-length': [0],
    }
}
