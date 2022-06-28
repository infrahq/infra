export const providers = [
  {
    name: 'Okta',
    kind: 'okta',
    available: true,
  },
  {
    name: 'Google',
    kind: 'google',
  },
  {
    name: 'Azure Active Directory',
    kind: 'azure',
  },
  {
    name: 'GitHub',
    kind: 'github',
  },
  {
    name: 'GitLab',
    kind: 'gitlab',
  },
  {
    name: 'OpenID',
    kind: 'oidc',
  },
]

export function kind(url) {
  if (url?.endsWith('.okta.com')) {
    return 'okta'
  }

  return ''
}
