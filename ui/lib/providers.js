export const providers = [
  {
    name: 'Azure Active Directory',
    kind: 'azure',
    available: true,
  },
  {
    name: 'Google',
    kind: 'google',
    available: true,
  },
  {
    name: 'Okta',
    kind: 'okta',
    available: true,
  },
  {
    name: 'OpenID',
    kind: 'oidc',
    available: true,
  },
  {
    name: 'GitHub',
    kind: 'github',
  },
  {
    name: 'GitLab',
    kind: 'gitlab',
  },
]

export const googleSocialLoginID = '45xp' // == 600613 (snowflake ID), the ID we reserve for the google social login provider
