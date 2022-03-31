module.exports = {
    extends: ['@commitlint/config-conventional'],
    rules: {
        'type-enum': [
            2,
            'always',
            [
                // feat is used for a new feature or enhancement and will increment the minor version.
                'feat',
                // fix is used for bug fixes and will increment the patch version.
                'fix',
                // improvement is used for quality improvements, refactors, and changes to docs.
                'improve',
                // maintain is used for maintenance tasks like updating dependencies,
                // releasing a new version, or reverting a change.
                'maintain',
            ]
        ],
        'body-max-line-length': [0],
    }
}
